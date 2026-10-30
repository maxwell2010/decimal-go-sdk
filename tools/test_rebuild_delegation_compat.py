import hashlib
import json
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import rebuild_delegation_compat as rebuild


class DelegationReproductionTests(unittest.TestCase):
    def test_escaped_binding_literal_has_exact_golden_output(self):
        raw = b'var Meta = bind.MetaData{ABI: "[{\\"type\\":\\"function\\",\\"name\\":\\"getStakes\\"}]",}\n'
        expected = b'[{"type":"function","name":"getStakes"}]\n'
        self.assertEqual(rebuild.render_binding(raw), expected)
        for invalid in [b'[]', b'ABI: "{}",', raw + raw, b'ABI: "[not JSON]",']:
            with self.subTest(invalid=invalid), self.assertRaises(ValueError):
                rebuild.render_binding(invalid)

    def test_verification_precedes_any_write_and_rejects_drift(self):
        raw = b'ABI: "[{\\"type\\":\\"function\\"}]",'
        rendered = rebuild.render_binding(raw)
        with tempfile.TemporaryDirectory() as tmp:
            source, dest = Path(tmp)/'binding.go', Path(tmp)/'abi.json'
            source.write_bytes(raw)
            dest.write_bytes(b'original destination')
            with self.assertRaisesRegex(ValueError, 'source does not match'):
                rebuild.rebuild(source, dest)
            self.assertEqual(dest.read_bytes(), b'original destination')
            with patch.object(rebuild, 'SOURCE_SHA256', hashlib.sha256(raw).hexdigest()):
                with self.assertRaisesRegex(ValueError, 'extracted ABI'):
                    rebuild.rebuild(source, dest)
                self.assertEqual(dest.read_bytes(), b'original destination')
                with patch.object(rebuild, 'ABI_SHA256', hashlib.sha256(rendered).hexdigest()):
                    with self.assertRaisesRegex(ValueError, 'differs'):
                        rebuild.rebuild(source, dest, check=True)
                    rebuild.rebuild(source, dest)
                    rebuild.rebuild(source, dest, check=True)
                    source.write_bytes(raw+b'\n')
                    with self.assertRaisesRegex(ValueError, 'source does not match'):
                        rebuild.rebuild(source, dest)
                    self.assertEqual(dest.read_bytes(), rendered)

    def test_pins_match_published_manifest(self):
        manifest = json.loads((rebuild.ROOT/'docs/DELEGATION_UPGRADE_SOURCES.json').read_text())
        entry = next(x for x in manifest['files'] if x['path']=='contracts/delegation/DecimalDelegation.go')
        self.assertEqual(manifest['commit'], rebuild.COMMIT)
        self.assertEqual(entry['source_sha256'], rebuild.SOURCE_SHA256)
        self.assertEqual(manifest['compatibility_abi']['sha256'], rebuild.ABI_SHA256)


if __name__ == '__main__':
    unittest.main()
