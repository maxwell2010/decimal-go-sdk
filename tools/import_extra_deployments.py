"""Import reviewed additional static-address deployment interfaces, offline.

Rejects role mismatches: a token ABI at an old Checks/GasCenter address is not
imported as a check or gas API. ABI presence alone is not bytecode equivalence.
"""
import hashlib
import json
from generate import ROOT, generate, normalize

def main():
    evidence=ROOT/'docs/deployments/testnet-2026-10-29-additional'
    metadata=json.loads((evidence/'metadata.json').read_text())
    if metadata['chain_id']!='202020' or metadata['block']!=22474733:
        raise SystemExit('Unexpected evidence snapshot: review before versioning')
    selected={
        'safe-static':('TestnetSafe20261029',{'getOwnersWithWeights','getWeightThreshold','setup','execTransaction'}),
        'safe-factory-static':('TestnetSafeFactory20261029',{'createProxyWithNonce'}),
        'multisend-static':('TestnetSafeMultiSend20261029',{'multiSend'}),
    }
    catalog=[];sources=[]
    for slug,(profile,required) in selected.items():
        rows=[e for e in metadata['entries'] if e['slug']==slug]
        if len(rows)!=1:raise SystemExit('Missing evidence entry')
        row=rows[0]
        if row.get('error') or not row.get('code_hash') or not row.get('address_source'):
            raise SystemExit('Incomplete static runtime/address evidence')
        raw=(evidence/(slug+'.abi.json')).read_bytes()
        if hashlib.sha256(raw).hexdigest()!=row['abi_sha256']:raise SystemExit('ABI checksum mismatch')
        abi=normalize(json.loads(raw))
        if not required.issubset({x.get('name') for x in abi if x.get('type')=='function'}):
            raise SystemExit('ABI does not expose the expected role')
        dest=ROOT/'abi'/f'{profile}.json';dest.write_text(json.dumps(abi,indent=2)+'\n',encoding='utf-8')
        methods=generate(profile,abi)
        for item in methods:
            item.update(source_status='versioned-public-testnet-API-artifact',chain_id='202020',observed_block=metadata['block'],target_proxy=row['address'])
        catalog+=methods
        sources.append({'profile':profile,'chain_id':'202020','observed_block':metadata['block'],
                        'target_proxy':row['address'],'address_source':row['address_source'],
                        'runtime_code_hash':row['code_hash'],'api_url':row['api_url'],
                        'source_abi_sha256':row['abi_sha256'],
                        'normalized_abi_sha256':hashlib.sha256(dest.read_bytes()).hexdigest(),
                        'status':'explicit versioned interface; no write/role acceptance; never substitute defaults'})
    (ROOT/'docs/additional-deployment-catalog.json').write_text(json.dumps(catalog,indent=2)+'\n')
    (ROOT/'docs/additional-deployment-sources.json').write_text(json.dumps({'evidence':'deployments/testnet-2026-10-29-additional/metadata.json','profiles':sources},indent=2)+'\n')
    print(f'{len(sources)} profiles, {len(catalog)} additional methods')

if __name__=='__main__':main()
