"""Generate explicit mainnet interfaces from reviewed read-only evidence, offline.

Official API ABI attribution is not cryptographic proof of every method. These
profiles preserve chain75 guards and do not replace defaults or certify writes.
"""
import hashlib
import json
from generate import ROOT, generate, normalize


def checked_artifact(directory, row):
    path = directory / row['abi_file']
    if path.parent != directory:
        raise ValueError('ABI must be directly inside the evidence directory')
    raw = path.read_bytes()
    if hashlib.sha256(raw).hexdigest() != row['abi_sha256']:
        raise ValueError('reviewed ABI checksum mismatch')
    implementation = row['implementation'].lower()
    if row['api_url'].lower() != 'https://api.decimalchain.com/api/v1/contracts/compact/' + implementation:
        raise ValueError('implementation API source mismatch')
    if not row.get('implementation_code_hash') or not row.get('proxy_code_hash'):
        raise ValueError('implementation/proxy code identity required')
    return normalize(json.loads(raw))


def main():
    catalog, sources = [], []
    directories = [ROOT/'docs/deployments/mainnet-2026-11-03-nft',
                   ROOT/'docs/deployments/mainnet-2026-11-03-token']
    for directory in directories:
        meta = json.loads((directory/'metadata.json').read_text(encoding='utf-8'))
        if str(meta['chain_id']) != '75' or not meta.get('block_hash'):
            raise ValueError('reviewed mainnet block identity required')
        rows = meta.get('profiles', [meta])
        for row in rows:
            role = row.get('role', 'nft')
            name, required = {
                'nft': ('MainnetNFTCenter20261103', {'createDRC721','createDRC1155','createDRC721Reserveless','createDRC1155Reserveless'}),
                'reserve': ('MainnetReserveToken20261103', {'updateMinTotalSupply','updateMaxTotalSupply'}),
                'reserveless': ('MainnetReservelessToken20261103', {'mint','cap','mintable'}),
            }[role]
            entries = checked_artifact(directory, row)
            if not required.issubset({x.get('name') for x in entries if x.get('type')=='function'}):
                raise ValueError('ABI does not expose the reviewed role')
            proxy = row.get('proxy', row.get('target_proxy'))
            if not proxy or proxy.lower() == row['implementation'].lower():
                raise ValueError('distinct execution proxy required')
            dest = ROOT/'abi'/(name+'.json')
            dest.write_bytes((json.dumps(entries,indent=2)+'\n').encode())
            records = generate(name,entries)
            for record in records:
                record.update(source_status='versioned-official-mainnet-API-interface',chain_id='75',observed_block=meta['block'],target_proxy=proxy)
            catalog.extend(records)
            sources.append({'profile':name,'chain_id':'75','observed_block':meta['block'],
                'block_hash':meta['block_hash'],'target_proxy':proxy,'abi_address':row['implementation'],
                'runtime_code_hash':row['implementation_code_hash'],'proxy_code_hash':row['proxy_code_hash'],
                'api_url':row['api_url'],'source_abi_sha256':row['abi_sha256'],
                'normalized_abi_sha256':hashlib.sha256(dest.read_bytes()).hexdigest(),
                'binding':row['binding'],'beacon':row.get('beacon'),'beacon_code_hash':row.get('beacon_code_hash'),
                'evidence':str(directory.relative_to(ROOT/'docs')).replace('\\','/')+'/metadata.json',
                'evidence_files_sha256':{p.name:hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(directory.glob('*.json'))},
                'status':'explicit chain75 interface and pinned proxy binding; API ABI is not cryptographic method proof; no write/role acceptance; never send to implementation'})
    (ROOT/'docs/mainnet-deployment-catalog.json').write_bytes((json.dumps(catalog,indent=2)+'\n').encode())
    (ROOT/'docs/mainnet-deployment-sources.json').write_bytes((json.dumps({'profiles':sources},indent=2)+'\n').encode())
    print(len(sources),'mainnet profiles,',len(catalog),'methods')


if __name__ == '__main__':
    main()
