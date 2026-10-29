"""Import implementation interfaces identified by TokenCenter view getters.

No network or signing. Candidate token/proxy binding remains explicit evidence,
not an assumption that any supplied address has this implementation.
"""
import hashlib
import json
from generate import ROOT,generate,normalize

def main():
    evidence=ROOT/'docs/deployments/testnet-2026-10-29-token-implementations'
    metadata=json.loads((evidence/'metadata.json').read_text())
    probes=json.loads((ROOT/'docs/deployments/testnet-2026-10-29-tokens/role-probes.json').read_text())
    beacons=json.loads((ROOT/'docs/deployments/testnet-2026-10-29-tokens/beacon-probes.json').read_text())
    if metadata['chain_id']!='202020' or metadata['block']!=22474834 or probes['block']!='22474794':raise SystemExit('Unexpected reviewed snapshot')
    selected={
      'reserve-implementation-candidate':('TestnetReserveToken20261029','implementation()',{'buy','sell','updateIdentity','updateMaxTotalSupply'}),
      'reserveless-implementation-candidate':('TestnetReservelessToken20261029','implementationReserveless()',{'mint','cap','mintable','burnable','owner'}),
    }
    catalog=[];sources=[]
    for slug,(profile,getter,required) in selected.items():
        rows=[e for e in metadata['entries'] if e['slug']==slug]
        if len(rows)!=1:raise SystemExit('Missing implementation evidence')
        row=rows[0]
        role='reserveless' if slug.startswith('reserveless-') else 'reserve'
        bindings=[b for b in beacons['entries'] if b['role']==role]
        if beacons['chain_id']!='202020' or beacons['block']!='22474794' or len(bindings)!=1:raise SystemExit('Missing reviewed beacon binding')
        binding=bindings[0]
        if binding.get('error') or binding['implementation'].lower()!=row['address'].lower() or binding['implementation_code_hash']!=row['code_hash']:raise SystemExit('Beacon implementation/runtime mismatch')
        observed=[p for p in probes['observations'] if p['Role']=='token-center' and p['Signature']==getter]
        if len(observed)!=1 or observed[0].get('error') or observed[0]['result'][0].lower()!=row['address'].lower():raise SystemExit('Registry getter does not identify implementation')
        if row.get('error') or not row.get('code_hash'):raise SystemExit('Missing implementation runtime')
        raw=(evidence/(slug+'.abi.json')).read_bytes()
        if hashlib.sha256(raw).hexdigest()!=row['abi_sha256']:raise SystemExit('Artifact checksum mismatch')
        abi=normalize(json.loads(raw))
        if not required.issubset({m.get('name') for m in abi if m.get('type')=='function'}):raise SystemExit('Wrong implementation interface')
        destination=ROOT/'abi'/f'{profile}.json';destination.write_text(json.dumps(abi,indent=2)+'\n',encoding='utf-8')
        methods=generate(profile,abi)
        for method in methods:method.update(source_status='versioned-public-testnet-implementation-interface',chain_id='202020',observed_block=metadata['block'])
        catalog+=methods
        sources.append({'profile':profile,'chain_id':'202020','observed_block':metadata['block'],'abi_address':row['address'],
          'runtime_code_hash':row['code_hash'],'api_url':row['api_url'],'source_abi_sha256':row['abi_sha256'],
          'normalized_abi_sha256':hashlib.sha256(destination.read_bytes()).hexdigest(),
          'registry_getter':getter,'registry_getter_block':probes['block'],'registry_getter_target':observed[0]['Address'],
          'target_proxy':binding['target'],'beacon':binding['beacon'],'beacon_code_hash':binding['beacon_code_hash'],'binding_block':beacons['block'],
          'binding_evidence':'deployments/testnet-2026-10-29-tokens/beacon-probes.json',
          'status':'explicit implementation interface with standard beacon storage/getter binding; API ABI not cryptographic interface proof; no writes; never send transactions to implementation'})
    (ROOT/'docs/token-deployment-catalog.json').write_text(json.dumps(catalog,indent=2)+'\n')
    (ROOT/'docs/token-deployment-sources.json').write_text(json.dumps({'profiles':sources},indent=2)+'\n')
    print(f'{len(sources)} token implementation profiles, {len(catalog)} methods')
if __name__=='__main__':main()
