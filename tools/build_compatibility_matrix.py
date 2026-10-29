"""Build an offline compatibility matrix from recorded, bounded RPC observations."""
import argparse
import hashlib
import json
from pathlib import Path
import re
from capture_compatibility_tests import REQUIRED, source_fingerprint

ROOT = Path(__file__).resolve().parents[1]
CATALOGS = ['contract-catalog.json', 'deployment-catalog.json',
            'additional-deployment-catalog.json', 'token-deployment-catalog.json']
MANIFESTS = ['deployment-sources.json', 'additional-deployment-sources.json', 'token-deployment-sources.json']
CHAINS = {'testnet': '202020', 'mainnet': '75'}
ROLES = {'ContractCenter': ['contract-center'], 'TokenCenter': ['token-center'],
         'NFTCenter': ['nft-center'], 'MasterValidator': ['master-validator'],
         'Checks': ['checks'], 'Delegation': ['delegation'], 'DelegationNFT': ['delegation-nft'],
         'Multicall': ['multicall'], 'Token': ['reserve-token', 'reserveless-token']}
HASH = re.compile(r'^0x[0-9a-fA-F]{64}$')
ADDRESS = re.compile(r'^0x[0-9a-fA-F]{40}$')


def read(path):
    return json.loads((ROOT / path).read_text(encoding='utf-8'))


def height(value):
    if isinstance(value, bool): raise ValueError('boolean block height')
    number = int(value, 16) if isinstance(value, str) and value.startswith('0x') else int(value)
    if number < 0 or number >= 1 << 64: raise ValueError('invalid block height')
    return number


def validate_snapshots(deployment, simulation, history, chain):
    if deployment['chain_id'] != chain or simulation['ChainID'] != chain or history['chain_id'] != chain:
        raise ValueError('network chain mismatch')
    if not HASH.fullmatch(deployment['block_hash']) or not simulation['Consistent']:
        raise ValueError('unverified snapshot identity')
    if height(deployment['block']) != simulation['Block'] or deployment['block_hash'].lower() != simulation['BlockHash'].lower():
        raise ValueError('deployment/simulation snapshots do not describe the same block')
    routes = [r['route'] for r in simulation['Observations']]
    if len(routes) != len(set(routes)):
        raise ValueError('duplicate simulation routes')
    for row in simulation['Observations']:
        if row['status'] not in {'eth_call-success', 'failed', 'not-tested'}:
            raise ValueError('unknown simulation status')
        if row['status'] == 'eth_call-success' and row['route'] == 'native-self-transfer-1-wei':
            if (not ADDRESS.fullmatch(row['From']) or int(row['From'], 16) == 0
                    or row['Target'].lower() != row['From'].lower() or row['Data'] != '0x'
                    or row['Value'] != '1' or int(row['ObservedBalance']) < 1 or row['ReturnData'] != '0x'):
                raise ValueError('native observation is not the declared positive self transfer')
    if history.get('error') or not HASH.fullmatch(history['to_block_hash']):
        raise ValueError('event query did not produce a valid bounded snapshot')
    first, last = height(history['from_block']), height(history['to_block'])
    if first > last or last - first >= 2000 or history['event_count'] != len(history['events']):
        raise ValueError('invalid event range/count')
    for event in history['events']:
        log = event['Log']
        if not first <= height(log['blockNumber']) <= last or log['address'].lower() != history['address'].lower() or log['removed']:
            raise ValueError('event outside its declared query')


def targets(profile, network, entries, manifest):
    source = manifest.get(profile)
    if source:
        if source['chain_id'] != CHAINS[network]: return 'not-applicable-network-profile', []
        address = source['target_proxy'].lower()
        return 'explicit-versioned-address-only', [r for r in entries if r.get('address', '').lower() == address]
    roles = ROLES.get(profile, [])
    return 'sampled-address-only-no-ABI-certification', [r for r in entries if r['role'] in roles and r.get('address')]


def simulation_matches(profile, signature, simulation, source, selected):
    # A self transfer at a versioned token address does not certify a pinned
    # Token ABI family, arbitrary recipients, approval/permit or other selectors.
    if signature != 'transfer(address,uint256)': return []
    matches = [r for r in simulation['Observations']
               if r['route'] == profile + '-self-transfer-1-raw' and r['status'] == 'eth_call-success']
    for row in matches:
        if not source or source['chain_id'] != simulation['ChainID']:
            raise ValueError('simulation has no matching reviewed network profile')
        owner = row['From']
        if not ADDRESS.fullmatch(owner) or int(owner, 16) == 0:
            raise ValueError('invalid simulated token account')
        expected_data = '0xa9059cbb' + '00' * 12 + owner[2:].lower() + '00' * 31 + '01'
        targets_at_snapshot = [r for r in selected if r['address'].lower() == row['Target'].lower()]
        references = targets_at_snapshot[0].get('references', []) if len(targets_at_snapshot) == 1 else []
        implementation_observed = any(r.get('address', '').lower() == row['Implementation'].lower()
                                      and r.get('code_hash', '').lower() == row['ImplementationHash'].lower() for r in references)
        beacon_observed = any(r.get('address', '').lower() == source['beacon'].lower()
                              and r.get('code_hash', '').lower() == source['beacon_code_hash'].lower() for r in references)
        if (row['Target'].lower() != source['target_proxy'].lower() or len(targets_at_snapshot) != 1
                or not implementation_observed or not beacon_observed
                or row['RuntimeHash'].lower() != targets_at_snapshot[0]['code_hash'].lower()
                or row['Implementation'].lower() != source['abi_address'].lower()
                or row['ImplementationHash'].lower() != source['runtime_code_hash'].lower()
                or row['Data'].lower() != expected_data or row['Value'] != '0'
                or int(row['ObservedBalance']) < 1 or row['ReturnData'].lower() != '0x' + '00' * 31 + '01'):
            raise ValueError('token simulation does not match its exact target, identity and self-transfer payload')
    return matches


def mainnet_checks_finding(testnet, mainnet):
    test = next((r for r in testnet['entries'] if r['role'] == 'checks' and r.get('address')), {})
    main = next((r for r in mainnet['entries'] if r['role'] == 'checks' and r.get('address')), {})
    implementations = {r.get('code_hash') for r in test.get('references', []) if r['kind'] == 'eip1967-implementation' and isinstance(r.get('code_hash'), str) and HASH.fullmatch(r['code_hash'])}
    probes = {r['signature']: r.get('result') for r in main.get('probes', [])}
    zero = '0x' + '00' * 32
    return bool(main.get('code_hash') in implementations and main.get('implementation_slot') == zero
                and main.get('beacon_slot') == zero and probes.get('owner()') == zero and probes.get('getImpl()') == zero)


def build():
    validation = read('docs/compatibility/local-validation.json')
    if validation['EVM_source_fingerprint_sha256'] != source_fingerprint():
        raise ValueError('source changed after recorded tests; rerun capture_compatibility_tests.py')
    if validation['modules']['.']['exit_code'] != 0 or any(validation['modules']['.']['required_suites'].get(n) != 'pass' for n in REQUIRED):
        raise ValueError('required local compatibility suites did not pass')
    catalog = [row for name in CATALOGS for row in read('docs/' + name)]
    if len({(r['profile'], r['signature']) for r in catalog}) != len(catalog):
        raise ValueError('duplicate ABI catalog records')
    manifest = {p['profile']: p for name in MANIFESTS for p in read('docs/' + name)['profiles']}
    methods, networks, deployments = [], {}, {}
    for network, chain in CHAINS.items():
        deployment = read('docs/compatibility/deployments-' + network + '.json')
        deployments[network] = deployment
        simulation = read('docs/compatibility/simulations-' + network + '-2026-10-29.json')
        history = read('docs/compatibility/checks-' + network + '.json')
        validate_snapshots(deployment, simulation, history, chain)
        probes = [p for entry in deployment['entries'] for p in entry.get('probes', [])]
        networks[network] = {
            'chain_id': chain, 'block': height(deployment['block']), 'rpc_block_hash': deployment['block_hash'],
            'rpc': deployment['rpc'], 'checked_utc': deployment['checked_utc'],
            'addresses_with_code': sum(r.get('code_bytes', 0) > 0 for r in deployment['entries']),
            'unresolved_registry_entries': [r['role'] for r in deployment['entries'] if not r.get('address')],
            'getter_returned_bytes': sum(bool(p.get('result')) and p.get('result') != '0x' for p in probes),
            'getter_failed_or_empty': sum(bool(p.get('error')) or p.get('result') == '0x' for p in probes),
            'simulation_cases': simulation['Observations'],
            'checks_history': {'count': history['event_count'], 'from_block': history['from_block'],
                               'to_block': history['to_block'], 'rpc_upper_block_hash': history['to_block_hash'],
                               'current_spendability': 'not-established', 'absence_scope': 'selected-window-only'},
            'network_transactions_sent': 0,
        }
        for row in catalog:
            status, selected = targets(row['profile'], network, deployment['entries'], manifest)
            getters = [{'address': entry['address'], **p} for entry in selected for p in entry.get('probes', [])
                       if p['signature'] == row['signature'] and row['method'].startswith('Read')]
            method_record = {k: row[k] for k in ['profile', 'signature', 'method', 'mutability']}
            methods.append({**method_record,
                            'ABI_source_observation': {k: v for k, v in row.items() if k not in method_record},
                            'network': network, 'chain_id': chain,
                            'snapshot_block': height(deployment['block']), 'rpc_block_hash': deployment['block_hash'],
                            'address_association': status,
                            'observed_targets': [{'address': r['address'], 'role': r['role'], 'code_hash': r.get('code_hash'),
                                                  'historical_hash_match': r.get('matches_historical_hash')} for r in selected],
                            'getter_observations': getters,
                            'simulation_cases': [r['route'] for r in simulation_matches(row['profile'], row['signature'], simulation, manifest.get(row['profile']), selected)],
                            'local_ABI_wire_suite': 'TestAllGeneratedABIWrappers', 'local_ABI_wire_status': 'recorded-pass',
                            'getter_caller_context': 'omitted-from-in-deployment-inspector; no global nonce semantics inferred',
                            'deployment_ABI_certification': 'not-certified',
                            'network_write_acceptance': ('not-tested-no-transaction-sent' if row['method'].startswith('Build')
                                                         else 'not-applicable-reader')})
    operations = [{**row, 'network_write_acceptance': 'not-tested-no-transaction-sent',
                   'boundary': 'related pinned builder does not certify every version or business workflow'}
                  for row in read('docs/operation-matrix.json')]
    receipts = read('docs/compatibility/checks-testnet-receipts.json')
    if receipts['chain_id'] != CHAINS['testnet'] or any(height(r['status']) != 1 for r in receipts['receipts']):
        raise ValueError('invalid saved Checks receipt')
    profiles = []
    for profile in sorted({r['profile'] for r in catalog}):
        rows = [r for r in catalog if r['profile'] == profile]
        profiles.append({'profile': profile, 'builders': sum(r['method'].startswith('Build') for r in rows),
                         'readers': sum(r['method'].startswith('Read') for r in rows),
                         'network': {network: {
                             'targets': len(next(r for r in methods if r['profile'] == profile and r['network'] == network)['observed_targets']),
                             'getter_signatures_returned_bytes': sum(any(p.get('result') and p['result'] != '0x' and not p.get('error')
                                                                        for p in r['getter_observations']) for r in methods
                                                                   if r['profile'] == profile and r['network'] == network),
                             'limited_simulated_signatures': sum(bool(r['simulation_cases']) for r in methods
                                                                if r['profile'] == profile and r['network'] == network),
                         } for network in CHAINS}})
    paths = ['docs/' + p for p in CATALOGS + MANIFESTS] + ['docs/operation-matrix.json']
    paths += ['docs/compatibility/' + p for p in ['deployments-testnet.json', 'deployments-mainnet.json',
              'simulations-testnet-2026-10-29.json', 'simulations-mainnet-2026-10-29.json',
              'checks-testnet.json', 'checks-mainnet.json', 'checks-testnet-receipts.json']]
    paths.append('docs/compatibility/local-validation.json')
    return {'schema': 1, 'scope': 'recorded read-only RPC and local test definitions; no transaction acceptance',
            'source_sha256': {p: hashlib.sha256((ROOT / p).read_bytes()).hexdigest() for p in paths},
            'summary': {'ABI_methods': len(catalog), 'ABI_builders': sum(r['method'].startswith('Build') for r in catalog),
                        'ABI_readers': sum(r['method'].startswith('Read') for r in catalog), 'profiles': len(profiles),
                        'reference_operations': len(operations), 'verified_saved_creation_receipts': len(receipts['receipts']),
                        'network_transactions_sent': 0, 'Cosmos_schema_count': 43,
                        'Cosmos_current_pinned_native_types': 8, 'Cosmos_live_acceptance': 'not-tested'},
            'findings': {'mainnet_checks_matches_testnet_implementation_with_zero_getters':
                         mainnet_checks_finding(deployments['testnet'], deployments['mainnet'])},
            'local_validation': validation, 'networks': networks, 'profiles': profiles, 'methods': methods, 'reference_operations': operations}


def markdown(data, russian=False):
    title = '# Матрица совместимости без отправки транзакций' if russian else '# Read-only compatibility matrix'
    intro = ('Это снимок наблюдений, а не процент готовности. Наличие кода, ответ getter или успешный eth_call не доказывают '
             'ABI целиком, полномочия пользователя, приём подписанной транзакции и settlement. Live-инспекторы не загружали ключи, не подписывали и не отправляли публичные транзакции. Отдельные локальные тесты используют синтетические векторы подписи.' if russian else
             'This is an observation matrix, not a readiness percentage. Runtime code, getter data and successful eth_call do not certify a whole ABI, user authorization, signed-transaction admission or settlement. Live inspectors did not load keys, sign or submit public-network transactions. Separate offline tests use synthetic signing vectors.')
    out = [title, '', intro, '', '[JSON matrix](compatibility/matrix.json) | [Deployments](DEPLOYMENT_COMPATIBILITY.md) | '
           '[Simulations](SIMULATION_COMPATIBILITY.md) | [Checks history](CHECKS_HISTORY.md)', '',
           '524 ABI methods / 260 builders / 264 readers / 20 profiles. 93 reference JS operations. ' +
           ('Локальный тест ABI: `TestAllGeneratedABIWrappers`, результат и fingerprint исходников: ' if russian else 'Local wire tests: `TestAllGeneratedABIWrappers`, recorded pass with a source fingerprint in ')
           + '[local-validation.json](compatibility/local-validation.json). [Validation](VALIDATION.md).', '',
           '| Network | Block | Code addresses | Getter responses / failures | Successful limited eth_call | Checks events in bounded window |',
           '|---|---|---|---|---|---|']
    for network, row in data['networks'].items():
        successes = sum(r['status'] == 'eth_call-success' for r in row['simulation_cases'])
        out.append(f"| {network} | {row['block']} | {row['addresses_with_code']} | {row['getter_returned_bytes']} / {row['getter_failed_or_empty']} | {successes} | {row['checks_history']['count']} |")
    warning = ('Инициализация и готовность Checks к операциям не подтверждены этими чтениями.' if russian else
               'Initialization and write readiness are not established by these reads.')
    if data['findings']['mainnet_checks_matches_testnet_implementation_with_zero_getters']:
        warning = ('**Mainnet Checks требует отдельной проверки deployment:** настроенный адрес имеет runtime testnet-implementation, нулевые slots/getImpl/owner. Это не подтверждённый initialized proxy. Адреса и конфигурация не изменялись.' if russian else
                   '**Mainnet Checks needs deployment review:** the configured address has the testnet implementation runtime and zero slots/getImpl/owner. An initialized usable proxy is not established. Configuration was not changed.')
    out += ['', warning, '',
            ('Успешные симуляции ограничены self-transfer: testnet reserve/reserveless token по 1 raw unit и mainnet DEL 1 wei. Они не проверяют внешнего получателя, burn, allowance, permit, права или bridge. Testnet native self-transfer не проверен: в выбранных 32 блоках не найден подходящий EOA. Контроль над публичными From-адресами не заявлен.' if russian else
             'Successful simulations cover self-transfer only: testnet reserve/reserveless token 1 raw unit and mainnet native DEL 1 wei. They do not certify external recipients, burns, allowance, permits, roles or bridges. Testnet native self-transfer was not tested because no suitable EOA was found in 32 selected blocks. Control of public From accounts is not claimed.'), '',
            ('Получены 6 testnet CreatedCheck и один отдельно проверенный receipt. Ноль mainnet-событий относится только к выбранному окну 2000 блоков. Историческое событие не доказывает текущую погашаемость. Время снимка истории отличается от снимка deployments; диапазоны и RPC-хеши сохранены раздельно.' if russian else
             'Six testnet CreatedCheck events and one separately verified receipt were observed. Zero mainnet events means only absence in the selected 2000-block window. Historical creation is not current spendability. History and deployment snapshots have distinct explicit ranges and RPC hashes.'), '',
            '| Profile | Builders / readers | Testnet targets / getter responses / simulated signatures | Mainnet targets / getter responses / simulated signatures |',
            '|---|---|---|---|']
    for p in data['profiles']:
        cells = [' / '.join(str(p['network'][n][k]) for k in ['targets', 'getter_signatures_returned_bytes', 'limited_simulated_signatures']) for n in CHAINS]
        out.append(f"| {p['profile']} | {p['builders']} / {p['readers']} | {cells[0]} | {cells[1]} |")
    out += ['', ('Targets означает наблюдённые адреса, а не доказанную привязку всего ABI. Versioned testnet-профили не переносятся на mainnet. Отсутствие выборки ERC721/1155 instances не заменяется чтением NFTCenter. Ответы getter сохранены как raw RPC bytes; нулевой owner не объявляется инициализацией. Methods и все 93 операции с независимыми статусами находятся в JSON.' if russian else
                  'Targets count observed addresses, not whole-ABI binding. Versioned testnet profiles are not adopted on mainnet. Unsampled ERC721/1155 instances are not replaced by NFTCenter observations. Getter responses are raw RPC bytes, and a zero owner is not initialization. Per-method records and all 93 operation statuses are in JSON.'), '',
            ('Осталось: авторитетные ABI недостающих поколений; state/proof/role inputs для других симуляций; отдельно разрешённая тестнет-приёмка. Для raw идентичности блока используйте Client.BlockIdentity, а не types.Header.Hash(). Before/after equality не является криптографическим state proof или гарантией финальности.' if russian else
             'Remaining work: authoritative interfaces for missing generations; valid state/proof/role inputs for other simulations; separately authorized funded-testnet acceptance. Use Client.BlockIdentity rather than types.Header.Hash() for the RPC block identity. Before/after equality is not a cryptographic state proof or finality guarantee.'), '']
    out += ['', ('В низкоуровневых getter-пробах инспектора поле From не задано. Такие результаты не подтверждают глобальность nonce или состояние конкретного создателя. Для caller-зависимых чтений используйте явно выбранный адрес и ReadFromAt после проверки ABI. Граница не меняется от наличия уже созданных чеков: без разрешённого proof погашение не симулируется.' if russian else 'Deployment getter probes omit From. Their results do not establish global nonce semantics or a particular creator account state. Use an explicit caller and ReadFromAt for reviewed caller-dependent getters. Existing creation hashes do not supply an authorized redemption proof.'), '']
    return '\n'.join(out)


def outputs():
    data = build()
    return {'docs/compatibility/matrix.json': json.dumps(data, indent=2) + '\n',
            'docs/COMPATIBILITY.md': markdown(data), 'docs/COMPATIBILITY.ru.md': markdown(data, True)}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    for name, content in outputs().items():
        target = ROOT / name
        if args.check:
            if not target.exists() or target.read_text(encoding='utf-8') != content:
                raise SystemExit('Compatibility matrix is stale: ' + name)
        else:
            target.write_text(content, encoding='utf-8', newline='\n')
    print('Compatibility matrix: 524 method records per network; no network write acceptance claimed')


if __name__ == '__main__': main()
