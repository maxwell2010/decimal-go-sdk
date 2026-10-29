"""Generate the operation matrix from pinned source text and the ABI catalog.
Run after generate.py: python tools/compare.py --js PATH --go PATH
"""
import argparse
import json
import hashlib
from pathlib import Path
import re
import subprocess
from generate import without_comments, signature

ROOT=Path(__file__).resolve().parents[1]
def sections(text,pattern):
    text=without_comments(text)
    matches=list(re.finditer(pattern,text,re.M))
    return {m.group(1):(text[m.start():matches[i+1].start() if i+1<len(matches) else len(text)],text[:m.start()].count('\n')+1) for i,m in enumerate(matches)}

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--js',type=Path,required=True);parser.add_argument('--go',type=Path,required=True);a=parser.parse_args()
    pins=json.loads((ROOT/'docs/sources.json').read_text())['pins']
    for repo,path,paths in [('dsc-js-sdk',a.js,['src/decimalevm/index.ts','src/decimalevm/call.ts']),('dsc-go-sdk',a.go,['decimalevm/methods'])]:
        head=subprocess.check_output(['git','-C',str(path),'rev-parse','HEAD'],text=True).strip()
        if head!=pins[repo]:raise SystemExit(f'{repo}: source revision differs from the pinned manifest')
        if subprocess.run(['git','-C',str(path),'diff','--quiet','HEAD','--',*paths],check=False).returncode:
            raise SystemExit(f'{repo}: source methods contain local changes; refusing source attribution')
    index=sections((a.js/'src/decimalevm/index.ts').read_text(),r'^  (?:public|private) (?:async )?(\w+)\(')
    calls=sections((a.js/'src/decimalevm/call.ts').read_text(),r'^[ \t]+(?:public|private) (?:async )?(\w+)\(')
    catalog=json.loads((ROOT/'docs/contract-catalog.json').read_text())
    go_names={r['name'].lower():r for r in json.loads((ROOT/'docs/upstream-inventory.json').read_text())['dsc-go-sdk']}
    go_alias={'createCollectionDRC721':'CreateNftCollection','createCollectionDRC1155':'CreateNftCollection',
              'approveForAllNFT':'SetApprovalForAllNFT','mintNFT':'MintNFT'}
    profiles={'delegation':'Delegation','delegationNft':'DelegationNFT','masterValidator':'MasterValidator','nftCenter':'NFTCenter','tokenCenter':'TokenCenter','checks':'Checks','multiCall':'Multicall'}
    records=[]
    pattern=r'(?:this\.(\w+)!\.contract|(?<!\.)\bcontract)(?:\.(?:estimateGas|callStatic))?(?:\.([A-Za-z]\w*)|\["([^"\n]+)"\])'
    for name,(body,line) in index.items():
        if not body.startswith('  public '):continue
        if not (363<=line<=1105 or 1812<=line<=1883):continue
        targets=re.findall(r'this\.call!\.(\w+)\(',body)
        methods=set();allowed=set();refs=[]
        for target in targets:
            if target not in calls: continue
            cb,cl=calls[target];refs.append({'method':target,'line':cl})
            for m in re.finditer(pattern,cb):
                # Estimation/static simulation targets can differ from the actual
                # submission target upstream. Never claim write parity from them.
                if '.estimateGas.' in m.group() or '.callStatic.' in m.group(): continue
                raw=m.group(2) or m.group(3)
                if raw in ('estimateGas','callStatic','interface','address','then'):continue
                if any(r['signature'].split('(')[0]==raw and r['method'].startswith('Read') for r in catalog) and not any(r['signature'].split('(')[0]==raw and r['method'].startswith('Build') for r in catalog):continue
                methods.add(raw.split('(')[0])
                if m.group(1): allowed.add(profiles.get(m.group(1),'deployment'))
                elif target in ('approveToken','transferToken','transferFromToken','burnToken','buyTokenForExactDEL','buyExactTokenForDEL','sellTokensForExactDEL','sellExactTokensForDEL','updateTokenIdentity','updateTokenMaxTotalSupply','updateTokenMinTotalSupply','permitToken'):
                    allowed.add('Token')
                elif target in ('setApprovalForAllNFT','approveNFT721','transferNFT','disableMintNFT','burnNFT','setTokenURINFT','mintNFT','mintNFTWithDELReserve','mintReserveless','mintNFTWithTokenReserve','addDELReserveNFT','addTokenReserveNFT','transferBatchNFT1155'):
                    allowed.update(('ERC721','ERC1155'))
                else: allowed.add('deployment')
            if target=='createCollection':
                collection_method={
                    'createCollectionDRC721':'createDRC721',
                    'createCollectionDRC1155':'createDRC1155',
                    'createCollectionDRC721Reserveless':'createDRC721Reserveless',
                    'createCollectionDRC1155Reserveless':'createDRC1155Reserveless',
                }[name]
                methods.add(collection_method);allowed.add('NFTCenter')
        if name in ('multiCall','multiSendToken'):methods.add('aggregate');allowed.add('Multicall')
        if name in ('sendDEL','burnDEL'):status='native';matches=[]
        elif name=='multiSendToken':status='DEL-only-token-route-disabled';matches=[]
        else:
            matches=[r for r in catalog if r['method'].startswith('Build') and r['signature'].split('(')[0] in methods and r['profile'] in allowed]
            status='pinned-ABI-builder' if matches else 'explicit-deployment-ABI-required'
        counterpart=go_names.get(go_alias.get(name,name).lower())
        records.append({'operation':name,'line':line,'go_counterpart':counterpart,'methods':sorted(methods),'profiles':sorted(allowed),
                        'status':status,'builders':[r['profile']+'.'+r['method'] for r in matches],
                        'call_source':refs,'deployed_validation':'not-run'})
    if len(records)!=93:raise SystemExit(f'Unexpected write-method count: {len(records)}')
    # Deployment evidence enriches provenance without promoting API interfaces to
    # pinned-node support or claiming successful on-chain transaction execution.
    evidence_dir=ROOT/'docs/deployments/testnet-2026-10-29'
    if (evidence_dir/'metadata.json').exists():
        evidence=json.loads((evidence_dir/'metadata.json').read_text())
        slug_profiles={'delegation':'Delegation','delegation-nft':'DelegationNFT','token-center':'TokenCenter','nft-center':'NFTCenter','master-validator':'MasterValidator'}
        for r in records:
            if r['status']!='explicit-deployment-ABI-required':continue
            artifacts=[]
            for item in evidence['entries']:
                slug=item['slug'].removesuffix('-implementation')
                file=evidence_dir/(item['slug']+'.abi.json')
                if not item.get('proxy') or item.get('error') or slug_profiles.get(slug) not in r['profiles'] or not file.exists():continue
                if hashlib.sha256(file.read_bytes()).hexdigest()!=item['abi_sha256']:raise SystemExit('Deployment evidence ABI hash mismatch')
                methods=[signature(m) for m in json.loads(file.read_text()) if m.get('type')=='function' and m['name'] in r['methods']]
                if methods:artifacts.append({'chain_id':evidence['chain_id'],'block':evidence['block'],'proxy':item['proxy'],'implementation':item['address'],'implementation_code_hash':item['code_hash'],'abi_sha256':item['abi_sha256'],'api_url':item['api_url'],'signatures':methods,'validation':'public-API-artifact-and-pinned-storage-only; no transaction executed'})
            if artifacts:r['deployment_abi_evidence']=artifacts
    (ROOT/'docs/operation-matrix.json').write_text(json.dumps(records,indent=2)+'\n')
    md=['# Official JS / Go operation matrix\n',
        'Pinned sources are recorded in [sources.json](sources.json). Regenerate with `tools/compare.py`.\n',
        '93 active EVM public write entry points are in the pinned JS index. The earlier regex count 94 included '
        '`mintNFT` inside a block comment. Block and line comments are now excluded. '
        'These are API entry points, not unique chain transaction types. The official Go methods directory has 58 exported '
        'functions including builders and signing helpers. Name matches are evidence of a counterpart, not semantic equivalence.\n',
        '`pinned-ABI-builder` means at least one related write signature has a typed builder; it does not mean all variants '
        'of that JS workflow are equivalent or deployed. Each builder accepts raw base units; JS may do approvals, conversions, '
        'permit signing, metadata extraction and other orchestration. `BuildReferenceOperation` uses ABI-level arguments, '
        'not the JS parameter order. Explicit deployment ABI is required for unmatched variants.\n',
        'Classification follows the actual submission contract, excluding `estimateGas`/`callStatic` targets. '
        'In particular, the three NFT reset workflows estimate against Delegation upstream but submit against '
        'DelegationNFT; coin reset signatures are not evidence of NFT reset coverage. Private methods do not '
        'extend the body of a preceding public method. `BuildReferenceOperationSignature` selects an exact '
        'overload while retaining the operation and profile restrictions.\n',
        '| JS operation | Official Go named counterpart | Our builders / route | Source |\n|---|---|---|---|\n']
    for r in records:
        link=f'https://bitbucket.org/decimalteam/dsc-js-sdk/src/6790d35e2decb0cbb06a9149a9c476c834f99223/src/decimalevm/index.ts#lines-{r["line"]}'
        old=r['go_counterpart'];oldname=old['name'] if old else 'not found by name/known aliases'
        route=', '.join(r['builders']) or r['status']
        if r.get('deployment_abi_evidence'):route+='; [testnet implementation ABI artifact](deployments/testnet-2026-10-29/README.md)'
        md.append(f'| `{r["operation"]}` | {oldname} | {route} | [JS]({link}) |\n')
    (ROOT/'docs/OPERATION_MATRIX.md').write_text(''.join(md),encoding='utf-8')
    stats={s:sum(r['status']==s for r in records) for s in sorted({r['status'] for r in records})}
    print(json.dumps({'JS_write_entries':len(records),'status':stats,'official_go_missing_name_or_alias':sum(r['go_counterpart'] is None for r in records)}))
if __name__=='__main__':main()
