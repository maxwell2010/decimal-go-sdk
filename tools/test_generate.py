import json
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest
from unittest.mock import patch
import generate
from generate import normalize, exported, verify_source_file

class ArtifactValidation(unittest.TestCase):
    @unittest.skipUnless(shutil.which('gofmt'), 'gofmt unavailable')
    def test_delegation_compatibility_models_are_reproducible(self):
        for profile in ('Delegation', 'DelegationNFT'):
            entries = json.loads((generate.ROOT / 'abi' / (profile + '.json')).read_text())
            original = json.dumps(entries)
            with tempfile.TemporaryDirectory() as tmp, patch.object(generate, 'ROOT', Path(tmp)):
                generate.generate(profile, entries)
                output = Path(tmp) / (profile.lower() + '_generated.go')
                subprocess.run(['gofmt', '-w', str(output)], check=True)
                rendered = output.read_text()
            expected = (generate.ROOT / (profile.lower() + '_generated.go')).read_text()
            self.assertEqual(rendered, expected)
            self.assertEqual(json.dumps(entries), original)
            self.assertEqual(rendered.count('`abi:"holdStartTime"`'), 6 if profile == 'Delegation' else 0)

    def test_wrapped_and_double_encoded(self):
        abi=[{'type':'function','name':'transfer','inputs':[{'name':'to','type':'address'}],'outputs':[]}]
        self.assertEqual(normalize(json.dumps({'abi':abi})),abi)
    def test_rejects_source_injection_identifiers(self):
        for name in ['bad"name','bad`name','bad\nname','123name','']:
            with self.assertRaises(ValueError):normalize([{'type':'function','name':name}])
    def test_nested_component_tags_are_validated(self):
        with self.assertRaises(ValueError):normalize([{'type':'function','name':'f','inputs':[{'name':'x','type':'tuple','components':[{'name':'bad`tag','type':'uint256'}]}]}])
    def test_underscore_fields_remain_exported(self):
        self.assertEqual(exported('_owner'),'Owner')
        self.assertEqual(exported('$owner'),'Owner')
        self.assertEqual(exported('___'),'Value')
        self.assertEqual(exported('_123'),'Value123')

    def test_types_cannot_inject_generated_source(self):
        for typ in ['bytes32]byte;var injected=true;//','uint999','bytes33','uint256[0]','uint256[-1]','uint256[01]','tuple','address payable']:
            with self.subTest(typ=typ),self.assertRaises(ValueError):
                normalize([{'type':'function','name':'f','inputs':[{'name':'v','type':typ}]}])
        for items in [None,{},['bad']]:
            with self.subTest(items=items),self.assertRaises(ValueError):
                normalize([{'type':'function','name':'f','inputs':items}])

    def test_untracked_and_modified_sources_are_rejected(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            def git(*args):
                return subprocess.run(['git','-C',tmp,*args],check=True,capture_output=True)
            git('init');git('config','user.email','fixture@example.invalid');git('config','user.name','fixture')
            file=root/'abi.json';file.write_text('[]')
            with self.assertRaises(subprocess.CalledProcessError): verify_source_file(root,'abi.json')
            git('add','abi.json');git('commit','-m','fixture')
            verify_source_file(root,'abi.json')
            file.write_text('[{}]')
            with self.assertRaises(subprocess.CalledProcessError): verify_source_file(root,'abi.json')
            # Index flags must not turn modified bytes into pinned source evidence.
            git('update-index','--assume-unchanged','abi.json')
            with self.assertRaises(ValueError): verify_source_file(root,'abi.json')
            git('update-index','--no-assume-unchanged','abi.json')
            git('update-index','--skip-worktree','abi.json')
            with self.assertRaises(ValueError): verify_source_file(root,'abi.json')

    def test_cross_path_tuple_type_collision_rejected(self):
        # Both paths generate FixtureFooArg0Result with incompatible fields.
        entries=[{'type':'function','name':'foo','stateMutability':'nonpayable','inputs':[
            {'name':'x','type':'tuple','components':[{'name':'result','type':'tuple','components':[{'name':'x','type':'bool'}]}]}],'outputs':[]},
            {'type':'function','name':'fooArg0','stateMutability':'view','inputs':[],'outputs':[{'name':'v','type':'uint256'}]}]
        with tempfile.TemporaryDirectory() as tmp,patch.object(generate,'ROOT',Path(tmp)):
            with self.assertRaisesRegex(ValueError,'conflicting generated Go type'): generate.generate('Fixture',entries)

    @unittest.skipUnless(shutil.which('go'),'Go compiler unavailable')
    def test_generated_identifier_collisions_compile(self):
        # Compile real generated wrappers against the SDK. Hidden tool fixtures
        # cannot be picked up by the regular recursive Go package scan.
        def fn(name,inputs):
            return {'type':'function','name':name,'inputs':inputs,'outputs':[],'stateMutability':'nonpayable'}
        entries=[fn('f',[]),fn('f',[{'name':'v','type':'uint256'}]),fn('f1',[]),
                 fn('_123',[{'name':'v','type':'function'}]),
                 fn('nested',[{'name':'v','type':'tuple','components':[
                     {'name':'a','type':'uint256'},{'name':'a2','type':'uint256'},
                     {'name':'a','type':'uint256'},{'name':'_123','type':'uint24[2][]'}]}])]
        with tempfile.TemporaryDirectory(dir=generate.ROOT,prefix='.generator-audit-') as tmp:
            root=Path(tmp)
            with patch.object(generate,'ROOT',root):generate.generate('Fixture',entries)
            (root/'stub.go').write_text('package decimal\nimport (sdk "github.com/maxwell2010/decimal-go-sdk"; "github.com/ethereum/go-ethereum/common")\ntype Contract = sdk.Contract\ntype Call = sdk.Call\ntype Client struct{}\nfunc (*Client) Contract(p string,a common.Address)(*Contract,error){return (&sdk.Client{}).Contract(p,a)}\n')
            result=subprocess.run(['go','test','.'],cwd=root,capture_output=True,text=True)
            self.assertEqual(result.returncode,0,result.stdout+result.stderr)

if __name__=='__main__':unittest.main()
