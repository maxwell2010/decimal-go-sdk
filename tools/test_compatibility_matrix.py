"""Do not promote codec, view or self-transfer observations to live acceptance."""
import copy
import json
import unittest
from unittest.mock import patch
from types import SimpleNamespace
import build_compatibility_matrix as matrix
import capture_compatibility_tests as capture


class CompatibilityMatrixTests(unittest.TestCase):
    def test_all_methods_and_operations_retain_independent_acceptance_status(self):
        data = matrix.build()
        self.assertEqual(data['summary']['ABI_methods'], 524)
        self.assertEqual(data['summary']['ABI_builders'], 260)
        self.assertEqual(len(data['methods']), 1048)
        self.assertEqual(len(data['reference_operations']), 93)
        self.assertEqual(data['summary']['network_transactions_sent'], 0)
        for row in data['methods']:
            self.assertEqual(row['deployment_ABI_certification'], 'not-certified')
            if row['method'].startswith('Build'):
                self.assertEqual(row['network_write_acceptance'], 'not-tested-no-transaction-sent')
        self.assertTrue(all(r['network_write_acceptance'] == 'not-tested-no-transaction-sent'
                            for r in data['reference_operations']))

    def test_versioned_profiles_and_simulation_scope_do_not_cross_networks(self):
        data = matrix.build()
        versioned = [r for r in data['methods'] if r['profile'].startswith('Testnet') and r['network'] == 'mainnet']
        self.assertTrue(versioned)
        self.assertTrue(all(r['address_association'] == 'not-applicable-network-profile'
                            and not r['observed_targets'] and not r['simulation_cases'] for r in versioned))
        simulated = [r for r in data['methods'] if r['simulation_cases']]
        self.assertEqual({r['profile'] for r in simulated},
                         {'TestnetReserveToken20261029', 'TestnetReservelessToken20261029'})
        self.assertTrue(all(r['signature'] == 'transfer(address,uint256)' and r['network'] == 'testnet' for r in simulated))
        self.assertEqual(data['networks']['testnet']['checks_history']['current_spendability'], 'not-established')
        self.assertEqual(data['networks']['mainnet']['checks_history']['absence_scope'], 'selected-window-only')

    def test_mixed_snapshot_or_missing_test_proof_is_rejected(self):
        dep = matrix.read('docs/compatibility/deployments-testnet.json')
        sim = matrix.read('docs/compatibility/simulations-testnet-2026-10-29.json')
        events = matrix.read('docs/compatibility/checks-testnet.json')
        matrix.validate_snapshots(dep, sim, events, '202020')
        for field, value in [('Block', sim['Block'] + 1), ('BlockHash', '0x' + '11' * 32),
                             ('Consistent', False), ('ChainID', '75')]:
            bad = copy.deepcopy(sim); bad[field] = value
            with self.subTest(field=field), self.assertRaises(ValueError):
                matrix.validate_snapshots(dep, bad, events, '202020')
        with patch.object(matrix, 'source_fingerprint', return_value='stale-source'):
            with self.assertRaisesRegex(ValueError, 'source changed'): matrix.build()

    def test_generated_matrix_and_guides_are_reproducible(self):
        for name, content in matrix.outputs().items():
            self.assertEqual((matrix.ROOT / name).read_text(encoding='utf-8'), content, name)
        data = json.loads((matrix.ROOT / 'docs/compatibility/matrix.json').read_text())
        for name, digest in data['source_sha256'].items():
            self.assertEqual(matrix.hashlib.sha256((matrix.ROOT / name).read_bytes()).hexdigest(), digest)

    def test_success_label_cannot_hide_wrong_simulation_payload_or_binding(self):
        dep = matrix.read('docs/compatibility/deployments-testnet.json')
        sim = matrix.read('docs/compatibility/simulations-testnet-2026-10-29.json')
        sources = matrix.read('docs/token-deployment-sources.json')['profiles']
        source = next(p for p in sources if p['profile'] == 'TestnetReserveToken20261029')
        _, selected = matrix.targets(source['profile'], 'testnet', dep['entries'], {source['profile']: source})
        self.assertEqual(len(matrix.simulation_matches(source['profile'], 'transfer(address,uint256)', sim, source, selected)), 1)
        for field, value in [('Target', '0x' + '11' * 20), ('From', '0x' + '00' * 20),
                             ('Data', '0x'), ('Value', '1'), ('ObservedBalance', '0'),
                             ('ReturnData', '0x' + '00' * 32), ('RuntimeHash', '0x' + '11' * 32),
                             ('ImplementationHash', '0x' + '22' * 32)]:
            bad = copy.deepcopy(sim)
            row = next(r for r in bad['Observations'] if r['route'] == source['profile'] + '-self-transfer-1-raw')
            row[field] = value
            with self.subTest(field=field), self.assertRaises(ValueError):
                matrix.simulation_matches(source['profile'], 'transfer(address,uint256)', bad, source, selected)
        changed = copy.deepcopy(selected)
        changed[0]['references'] = []
        with self.assertRaises(ValueError):
            matrix.simulation_matches(source['profile'], 'transfer(address,uint256)', sim, source, changed)

    def test_concurrent_source_change_cannot_record_a_test_pass(self):
        stdout = '\n'.join(json.dumps({'Action': 'pass', 'Test': name}) for name in capture.REQUIRED + capture.COSMOS_REQUIRED)
        result = SimpleNamespace(returncode=0, stdout=stdout, stderr='')
        with patch.object(capture, 'source_fingerprint', side_effect=['before', 'after']), \
                patch.object(capture.subprocess, 'run', return_value=result):
            with self.assertRaisesRegex(SystemExit, 'Source changed'): capture.main()

    def test_mainnet_checks_warning_is_derived_from_observations(self):
        testnet = matrix.read('docs/compatibility/deployments-testnet.json')
        mainnet = matrix.read('docs/compatibility/deployments-mainnet.json')
        self.assertTrue(matrix.mainnet_checks_finding(testnet, mainnet))
        changed = copy.deepcopy(mainnet)
        row = next(r for r in changed['entries'] if r['role'] == 'checks')
        row['code_hash'] = '0x' + '33' * 32
        self.assertFalse(matrix.mainnet_checks_finding(testnet, changed))


if __name__ == '__main__': unittest.main()
