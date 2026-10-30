"""Record actual offline Go test outcomes for the compatibility matrix."""
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import subprocess

ROOT = Path(__file__).resolve().parents[1]
REQUIRED = ['TestAllGeneratedABIWrappers', 'TestEmbeddedABIProvenance', 'TestVersionedABIProvenance',
            'TestChecksSavedOfficialReceipt', 'TestBlockIdentityUsesRPCProvidedHashAndStrictShape',
            'TestLocalEVMFixtureProvenance', 'TestLocalEVMOfficialHistoricalTokenLifecycle',
            'TestLocalEVMOfficialChecksCreationAndSigningProfileMismatch',
            'TestLocalEVMHistoricalChecksArtifactRedemptionAndExpiry',
            'TestLocalEVMOfficialWeightedSafeFactoryAndExecution', 'TestLocalEVMStandardPermitProvenance',
            'TestLocalEVMStandardPermitLifecycle', 'TestLocalEVMHistoricalChecksTokenBatchRedemptionAtomicity',
            'TestLocalChainSignedTransactionEnvelopesAndFees', 'TestLocalChainRevertedReceiptConsumesNonceAndFee',
            'TestLocalEVMObservedCurrentChecksRuntimeSigning',
            'TestLocalEVMObservedDecimalReservelessPermit', 'TestLocalEVMWeightedSafeOwnerAdministrationAndAuthorization',
            'TestLocalEVMCurrentChecksTokenBatchRedemptionAtomicity', 'TestLocalEVMCurrentChecksTokenByStandardPermit',
            'TestLocalEVMCurrentChecksExpiryRejectsRecipientProof', 'TestContractReadPreservesCancellationOnBackendResponse',
            'TestContractReadDynamicBytesDoNotAliasBackendBuffer',
            'TestSimulationReadCancellationAndOwnership', 'TestSimulationReadKeepsCustomError',
            'TestLocalEVMReserveEconomyLinearLifecycle', 'TestLocalEVMReserveEconomyRollbackAndLimits',
            'TestLocalEVMReserveDustExactOutputRejectsZeroInput', 'TestLocalEVMReserveDustPositiveBudgetFollowsFloorQuote',
            'TestLocalReserveOracleCRR100IntegerRounding', 'TestLocalReserveOracleCRR50PerfectSquareVectors',
            'TestLocalEVMReserveNonlinearQuoteExecutionConcordance', 'TestLocalEVMReserveExactOutputNonlinearConcordance',
            'TestLocalEVMReserveInitializationBoundaries', 'TestLocalEVMReserveAdministrationBoundaries',
            'TestLocalEVMReserveCurrentPriceRationalVectors', 'TestLocalEVMReserveQuoteOverloadsMatchState',
            'TestLocalEVMReserveBeforeQuotesCreditedBalanceContext', 'TestLocalReserveSignedChainEconomyAndActualFees',
            'TestDelegationStakeCompatibilityLayouts', 'TestDelegationStakeCompatibilityArraysAndMalformed',
            'TestDelegationStakeCompatibilityTypedReaders', 'TestDelegationUpgradeABIProvenance',
            'TestDelegationUpgradeEventsKeepOriginalLayouts',
            'TestDelegationUpgradeEventsRejectBadPaddingAndWrongProfile', 'TestDelegationSavedMainnetObservations',
            'TestUnifiedNFTDelegationOfflineBuildersAndOperator', 'TestUnifiedNFTDelegationSavedViews',
            'TestRetiredMainnetNFTDepositsFailOffline', 'TestDelegationReceiptPreservesRepeatedExitPairing']


def source_fingerprint(root=ROOT):
    paths = []
    for parent, directories, files in os.walk(root):
        directories[:] = [d for d in directories if d not in {'.tools', '.git', 'cosmos', 'dist', '__pycache__'}
                          and not d.startswith('.generator-audit-')]
        paths += [Path(parent) / n for n in files if n.endswith('.go')]
    paths += list((root / 'abi').glob('*.json'))
    paths += list((root / 'delegation_compat').glob('*.json'))
    paths += [root / 'go.mod', root / 'go.sum']
    paths += [root / 'docs' / n for n in ['contract-catalog.json', 'deployment-catalog.json',
              'additional-deployment-catalog.json', 'token-deployment-catalog.json',
              'deployment-sources.json', 'additional-deployment-sources.json', 'token-deployment-sources.json',
              'checks-js-vectors.json']]
    paths.append(root / 'docs/compatibility/checks-testnet-receipts.json')
    paths += [p for p in (root / 'testdata/evm').glob('*') if p.is_file()]
    paths += [p for p in (root / 'testdata/delegation').glob('*.json')]
    paths.append(root / 'docs/DELEGATION_UPGRADE_SOURCES.json')
    h = hashlib.sha256()
    for path in sorted(set(paths), key=lambda p: p.relative_to(root).as_posix()):
        h.update(path.relative_to(root).as_posix().encode() + b'\0' + path.read_bytes() + b'\0')
    return h.hexdigest()


def main():
    before = source_fingerprint()
    modules = {}
    for module in ['.', 'cosmos']:
        command = ['go', 'test', '-json', './...']
        result = subprocess.run(command, cwd=ROOT / module, capture_output=True, text=True)
        if result.returncode:
            print(result.stdout, result.stderr)
            raise SystemExit('Go tests failed; validation artifact not updated')
        events = [json.loads(line) for line in result.stdout.splitlines()]
        passed = [e for e in events if e.get('Action') == 'pass' and 'Test' in e]
        suites = {e['Test'] for e in passed if '/' not in e['Test']}
        if module == '.' and not set(REQUIRED).issubset(suites):
            raise SystemExit('Required compatibility suites did not execute')
        modules[module] = {'command': 'go test -json ./...', 'exit_code': 0,
                           'passing_test_events': len(passed), 'top_level_entries': len(suites),
                           'required_suites': {name: 'pass' for name in REQUIRED} if module == '.' else {}}
    after = source_fingerprint()
    if before != after:
        raise SystemExit('Source changed while tests ran; validation artifact not updated')
    data = {'checked_utc': datetime.now(timezone.utc).isoformat(),
            'go_version': subprocess.check_output(['go', 'version'], cwd=ROOT, text=True).strip(),
            'EVM_source_fingerprint_sha256': after, 'modules': modules,
            'boundary': 'offline tests, in-process EVM fixtures, isolated Ethereum dev-chain execution and mocked transports; no public transaction submission or settlement proof'}
    target = ROOT / 'docs/compatibility/local-validation.json'
    target.write_text(json.dumps(data, indent=2) + '\n', encoding='utf-8', newline='\n')
    print(json.dumps({m: v['passing_test_events'] for m, v in modules.items()}))


if __name__ == '__main__': main()
