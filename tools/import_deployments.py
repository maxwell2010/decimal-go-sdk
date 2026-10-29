"""Generate versioned typed profiles from reviewed public deployment evidence.

No network access. Run after cmd/abi-inspect and review metadata first. These
profiles are separate from the pinned node/JS catalog and never become defaults.
"""
import argparse
import hashlib
import json
from pathlib import Path
from generate import ROOT, generate, normalize

PROFILES={
    'contract-center-implementation':'TestnetContractCenter20261029',
    'token-center-implementation':'TestnetTokenCenter20261029',
    'nft-center-implementation':'TestnetNFTCenter20261029',
    'master-validator-implementation':'TestnetMasterValidator20261029',
}

def main():
    parser=argparse.ArgumentParser();parser.add_argument('--evidence',type=Path,default=ROOT/'docs/deployments/testnet-2026-10-29');args=parser.parse_args()
    metadata=json.loads((args.evidence/'metadata.json').read_text())
    if metadata['chain_id']!='202020' or metadata['block']!=22474456:raise SystemExit('Unexpected deployment evidence snapshot; review and version importer before importing')
    catalog=[];sources=[]
    for slug,profile in PROFILES.items():
        matches=[x for x in metadata['entries'] if x['slug']==slug]
        if len(matches)!=1:raise SystemExit(f'Missing/duplicate evidence: {slug}')
        entry=matches[0]
        if entry.get('error') or not entry.get('proxy') or not entry.get('code_hash'):raise SystemExit(f'Incomplete implementation evidence: {slug}')
        proxy=[x for x in metadata['entries'] if x['address']==entry['proxy'] and not x.get('proxy')]
        if len(proxy)!=1 or not proxy[0].get('code_hash'):raise SystemExit('Missing proxy runtime evidence')
        file=args.evidence/(slug+'.abi.json');raw=file.read_bytes()
        if hashlib.sha256(raw).hexdigest()!=entry['abi_sha256']:raise SystemExit('ABI artifact checksum mismatch')
        abi=normalize(json.loads(raw));dest=ROOT/'abi'/f'{profile}.json';dest.write_text(json.dumps(abi,indent=2)+'\n',encoding='utf-8')
        methods=generate(profile,abi)
        for method in methods:method['source_status']='versioned-public-testnet-API-artifact';method['chain_id']=metadata['chain_id'];method['observed_block']=metadata['block'];method['target_proxy']=entry['proxy']
        catalog.extend(methods)
        sources.append({'profile':profile,'chain_id':metadata['chain_id'],'observed_block':metadata['block'],'checked_utc':metadata['checked_utc'],
                        'target_proxy':entry['proxy'],'proxy_code_hash':proxy[0]['code_hash'],'implementation':entry['address'],'implementation_code_hash':entry['code_hash'],
                        'api_url':entry['api_url'],'source_abi_sha256':entry['abi_sha256'],'normalized_abi_sha256':hashlib.sha256(dest.read_bytes()).hexdigest(),
                        'status':'versioned public API artifact; not pinned node ABI; no write/role/settlement validation; do not substitute implementation as transaction target'})
    (ROOT/'docs/deployment-catalog.json').write_text(json.dumps(catalog,indent=2)+'\n')
    (ROOT/'docs/deployment-sources.json').write_text(json.dumps({'evidence':'deployments/testnet-2026-10-29/metadata.json','profiles':sources},indent=2)+'\n')
    print(f'{len(sources)} versioned testnet profiles, {len(catalog)} methods, separate from pinned contract-catalog.json')

if __name__=='__main__':main()
