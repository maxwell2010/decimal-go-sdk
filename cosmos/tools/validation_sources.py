"""Record the exact upstream ValidateBasic policies, constants and app denom override."""
import argparse, hashlib, json, pathlib, re, subprocess, zipfile
p=argparse.ArgumentParser();p.add_argument('--node',type=pathlib.Path,required=True);p.add_argument('--sdk',type=pathlib.Path,required=True);a=p.parse_args()
root=pathlib.Path(__file__).resolve().parents[1]
commit='9e6c6d718d662083c4a524376a66d2c50bd4bc77'
if subprocess.check_output(['git','-C',str(a.node),'rev-parse','HEAD'],text=True).strip()!=commit:raise SystemExit('wrong node commit')
paths=['x/coin/types/msg.go','x/fee/types/tx.go','x/legacy/types/msg.go','x/multisig/types/msg.go','x/nft/types/msg.go','x/swap/types/msg.go','x/validator/types/msg.go','x/validator/types/msg_emergency.go','x/coin/config/config.go','x/nft/config/config.go','x/multisig/types/config.go','app/denom_whitelist.go','types/utils.go']
subprocess.run(['git','-C',str(a.node),'diff','--quiet','HEAD','--',*paths],check=True)
records=[];inputs=[]
sdk_module=json.loads(subprocess.check_output(['go','mod','download','-json','github.com/cosmos/cosmos-sdk@v0.46.16'],text=True))
if a.sdk.resolve()!=pathlib.Path(sdk_module['Dir']).resolve():raise SystemExit('SDK must be verified module cache v0.46.16')
sdk_archive=zipfile.ZipFile(sdk_module['Zip'])
for rel in paths+['x/upgrade/types/msgs.go','x/upgrade/types/plan.go','types/coin.go','types/decimal.go']:
    sdk=rel not in paths;file=(a.sdk if sdk else a.node)/rel;text=file.read_text(encoding='utf-8');digest=hashlib.sha256(file.read_bytes()).hexdigest()
    if sdk and file.read_bytes()!=sdk_archive.read('github.com/cosmos/cosmos-sdk@v0.46.16/'+rel):raise SystemExit('modified SDK source '+rel)
    url=f'https://github.com/cosmos/cosmos-sdk/blob/v0.46.16/{rel}' if sdk else f'https://bitbucket.org/decimalteam/go-smart-node/src/{commit}/{rel}'
    inputs.append({'path':rel,'repository':'cosmos-sdk-v0.46.16' if sdk else 'go-smart-node','sha256':digest,'url':url})
    for match in re.finditer(r'func\s+\(\w+\s+\*?(Msg\w+)\)\s+ValidateBasic\(\) error',text):
        module='upgrade' if sdk else rel.split('/')[1]
        type_url=f'/cosmos.upgrade.v1beta1.{match[1]}' if sdk else f'/decimal.{module}.v1.{match[1]}'
        line=text[:match.start()].count('\n')+1
        records.append({'type_url':type_url,'path':rel,'line':line,'url':url+f'#L{line}' if sdk else url+f'#lines-{line}','sha256':digest})
inventory=json.loads((root/'schema/inventory.json').read_text())
if {r['type_url'] for r in records}!=set(inventory):raise SystemExit('ValidateBasic inventory mismatch')
whitelist_text=(a.node/'app/denom_whitelist.go').read_text(encoding='utf-8').split('var invalidDelegatedDenoms',1)[1]
whitelist=[json.loads(line.strip().removesuffix(',')) for line in whitelist_text.splitlines() if line.strip().startswith('"')]
(root/'schema/validation-denom-whitelist.json').write_text(json.dumps(whitelist,ensure_ascii=True,indent=2)+'\n',encoding='utf-8')
(root/'schema/validation-sources.json').write_text(json.dumps({'node_commit':commit,'sdk_version':'v0.46.16','scope':'Valid-message stateless acceptance predicates; no authorization/state/ante success; malformed custom integers fail closed','policies':sorted(records,key=lambda r:r['type_url']),'inputs':inputs},indent=2)+'\n',encoding='utf-8')
print(f'{len(records)} validation policies; {len(whitelist)} pinned app denomination exceptions')
