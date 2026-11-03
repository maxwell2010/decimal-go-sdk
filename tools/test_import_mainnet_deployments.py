import json
from pathlib import Path
import shutil
import tempfile
import unittest

import import_mainnet_deployments as importer


class MainnetArtifactChecks(unittest.TestCase):
    def test_reviewed_artifact_and_rejected_source_drift(self):
        original = importer.ROOT/'docs/deployments/mainnet-2026-11-03-nft'
        row = json.loads((original/'metadata.json').read_text(encoding='utf-8'))
        with tempfile.TemporaryDirectory() as tmp:
            directory = Path(tmp)
            dest = directory/row['abi_file']
            shutil.copy2(original/row['abi_file'], dest)
            entries = importer.checked_artifact(directory, row)
            self.assertIn('createDRC721Reserveless', {x.get('name') for x in entries})
            for changed in [dict(row, api_url='https://example.invalid/abi'),
                            dict(row, implementation_code_hash=''),
                            dict(row, abi_file='../outside.json')]:
                with self.subTest(changed=changed.get('api_url')), self.assertRaises(ValueError):
                    importer.checked_artifact(directory, changed)
            dest.write_bytes(dest.read_bytes()+b' ')
            with self.assertRaisesRegex(ValueError, 'checksum'):
                importer.checked_artifact(directory, row)


if __name__ == '__main__':
    unittest.main()
