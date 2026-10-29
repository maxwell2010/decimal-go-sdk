import copy
import json
from pathlib import Path
import tempfile
import unittest
from update_latest import REPOSITORY, release_manifest, update


def release(version='v0.1.0-preview.2'):
    stem = 'decimal-go-sdk-' + version[1:]
    return {'draft': False, 'tag_name': version, 'html_url': REPOSITORY + '/releases/tag/' + version,
            'assets': [{'name': name, 'state': 'uploaded', 'size': 10,
                        'browser_download_url': REPOSITORY + '/releases/download/' + version + '/' + name}
                       for name in [stem + '.zip', stem + '.tar.gz', 'SHA256SUMS']]}


class ChannelTests(unittest.TestCase):
    def test_only_complete_published_release_can_advance_channel(self):
        good = release()
        for field, value in [('draft', True), ('tag_name', 'other'), ('html_url', 'https://example.invalid')]:
            bad = copy.deepcopy(good); bad[field] = value
            with self.subTest(field=field), self.assertRaises(ValueError):
                release_manifest('v0.1.0-preview.2', bad)
        for assets in [[], good['assets'][:-1], good['assets'] + [good['assets'][0]]]:
            bad = copy.deepcopy(good); bad['assets'] = assets
            with self.assertRaises(ValueError): release_manifest('v0.1.0-preview.2', bad)
        for field, value in [('state', 'new'), ('size', 0), ('browser_download_url', 'https://example.invalid')]:
            bad = copy.deepcopy(good); bad['assets'][0][field] = value
            with self.assertRaises(ValueError): release_manifest('v0.1.0-preview.2', bad)

    def test_versions_and_fixed_module_paths(self):
        for version in ['v2.0.0', 'v0.01.0', 'v0.1.0-01', 'v0.1.0;echo bad', '../main', 'v0.1.0+local']:
            with self.subTest(version=version), self.assertRaises(ValueError): release_manifest(version, release(version))
        data = release_manifest('v0.1.0-preview.2', release())
        self.assertEqual(data['evm'], 'github.com/maxwell2010/decimal-go-sdk@v0.1.0-preview.2')
        self.assertEqual(data['cosmos'], 'github.com/maxwell2010/decimal-go-sdk/cosmos@v0.1.0-preview.2')

    def test_failed_update_preserves_previous_channel(self):
        with tempfile.TemporaryDirectory() as tmp:
            destination = Path(tmp) / 'latest.json'
            destination.write_text('previous')
            bad = release(); bad['draft'] = True
            with self.assertRaises(ValueError): update('v0.1.0-preview.2', bad, destination)
            self.assertEqual(destination.read_text(), 'previous')
            expected = update('v0.1.0-preview.2', release(), destination)
            self.assertEqual(json.loads(destination.read_text()), expected)
            self.assertFalse(destination.with_suffix('.json.tmp').exists())


if __name__ == '__main__': unittest.main()
