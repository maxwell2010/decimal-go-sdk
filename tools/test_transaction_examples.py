"""Regression checks for complete executable transaction documentation."""
import json
from pathlib import Path
import re
import subprocess
import sys
import unittest

sys.path.insert(0, str(Path(__file__).resolve().parent))
import generate_transaction_examples as examples


class TransactionExamplesTest(unittest.TestCase):
    def test_generated_artifacts_match_every_catalogued_builder(self):
        outputs, count, profile_count = examples.render()
        expected = []
        for catalog in examples.CATALOGS:
            expected.extend(row for row in json.loads((examples.ROOT/'docs'/catalog).read_text(encoding='utf-8'))
                            if row['method'].startswith('Build'))
        self.assertEqual(count, len(expected))
        self.assertEqual(profile_count, len({row['profile'] for row in expected}))
        for row in expected:
            source = outputs[f'examples/transactions/{row["profile"].lower()}.go']
            self.assertEqual(source.count('contract.'+row['method']+'('), 1)
            for filename in ('docs/TRANSACTIONS.md', 'docs/TRANSACTIONS.ru.md'):
                self.assertIn('`'+row['signature']+'`', outputs[filename])
                self.assertIn('contract.'+row['method']+'(', outputs[filename])
        for filename in ('docs/TRANSACTIONS.md', 'docs/TRANSACTIONS.ru.md'):
            self.assertEqual(len(re.findall(r'^### `Build', outputs[filename], re.M)), count)
        for filename, content in outputs.items():
            self.assertEqual((examples.ROOT/filename).read_text(encoding='utf-8'), content, filename)

    def test_fixture_uses_named_nested_tuple_and_fixed_array(self):
        value = examples.fixture({'type':'tuple[2]', 'components':[
            {'name':'owner','type':'address'}, {'name':'weights','type':'uint256[]'}]}, 'ProfileExampleArg0')
        self.assertTrue(value.startswith('[2]decimal.ProfileExampleArg0{'))
        self.assertEqual(value.count('Owner: fixtureAddress'), 2)
        self.assertEqual(value.count('Weights: []*big.Int{big.NewInt(1)}'), 2)

    def test_check_command(self):
        subprocess.run([sys.executable, '-B', str(Path(examples.__file__)), '--check'], check=True,
                       stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True)


if __name__ == '__main__':
    unittest.main()
