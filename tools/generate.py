"""Import pinned official ABIs and generate typed offline builders/readers.

Usage: python tools/generate.py --node PATH --js PATH --go PATH
Only the destination gosdk directory is modified. Source trees are read-only.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re
import subprocess

ROOT = Path(__file__).resolve().parents[1]
PINS = {'go-smart-node': '9e6c6d718d662083c4a524376a66d2c50bd4bc77',
        'dsc-js-sdk': '6790d35e2decb0cbb06a9149a9c476c834f99223',
        'dsc-go-sdk': '3ef4a089b6020889e60783c2026df5725fb960e2'}

def normalize(obj):
    while not isinstance(obj, list):
        obj = json.loads(obj) if isinstance(obj, str) else obj['abi']
    def identifier(value, empty=False):
        if not isinstance(value,str) or (value=='' and not empty) or (value and not re.fullmatch(r'[A-Za-z_$][A-Za-z0-9_$]*',value)):
            raise ValueError('invalid ABI identifier')
    def arguments(items):
        if not isinstance(items,list): raise ValueError('ABI arguments must be a list')
        for arg in items:
            if not isinstance(arg,dict): raise ValueError('invalid ABI argument')
            identifier(arg.get('name',''),empty=True)
            typ=arg.get('type')
            if not isinstance(typ,str) or not re.fullmatch(r'(?:address|bool|string|bytes(?:[1-9]|[12][0-9]|3[0-2])?|u?int(?:8|16|24|32|40|48|56|64|72|80|88|96|104|112|120|128|136|144|152|160|168|176|184|192|200|208|216|224|232|240|248|256)|tuple|function)(?:\[(?:[1-9][0-9]*)?\])*',typ):
                raise ValueError('unsupported or invalid ABI type')
            if typ.split('[')[0]=='tuple': arguments(arg.get('components'))
            elif 'components' in arg: raise ValueError('components require a tuple')
    for entry in obj:
        if not isinstance(entry,dict): raise ValueError('invalid ABI entry')
        if entry.get('type') in ('function','event','error'): identifier(entry.get('name'))
        arguments(entry.get('inputs',[]));arguments(entry.get('outputs',[]))
    return obj

def exported(name):
    s = re.sub('[^A-Za-z0-9_]', '_', name).lstrip('_')
    if s and s[0].isdigit(): s='Value'+s
    return s[:1].upper() + s[1:] if s else 'Value'

def unique_name(name, used, suffix=1):
    candidate=name
    while candidate in used:
        candidate=name+str(suffix);suffix+=1
    used.add(candidate)
    return candidate

def verify_source_file(root, relative):
    # git diff does not reject an untracked file. Both checks are required before
    # assigning a pinned commit URL to working-tree bytes.
    subprocess.run(['git','-C',str(root),'ls-files','--error-unmatch','--',relative],check=True,stdout=subprocess.DEVNULL)
    subprocess.run(['git','-C',str(root),'diff','--quiet','HEAD','--',relative],check=True)
    expected=subprocess.check_output(['git','-C',str(root),'rev-parse','HEAD:'+relative],text=True).strip()
    actual=subprocess.check_output(['git','-C',str(root),'hash-object','--path='+relative,str(Path(root).resolve()/relative)],text=True).strip()
    if actual != expected: raise ValueError('source content differs from pinned HEAD blob')

def without_comments(text):
    # Preserve positions and leave URL literals untouched.
    return re.sub(r'/\*[\s\S]*?\*/|^[ \t]*//[^\n]*',
                  lambda m: ''.join('\n' if c=='\n' else ' ' for c in m.group()),text,flags=re.M)

def signature_type(x):
    return ('(' + ','.join(signature_type(c) for c in x['components']) + ')' + x['type'][5:]
            if x['type'].startswith('tuple') else x['type'])

def signature(m):
    return m['name'] + '(' + ','.join(signature_type(x) for x in m['inputs']) + ')'

def register_struct(structs, name, declaration):
    if name in structs and structs[name] != declaration:
        raise ValueError('conflicting generated Go type: '+name)
    structs[name]=declaration

def go_type(x, prefix, structs):
    t = x['type']
    if t.endswith(']'):
        i = t.rindex('[')
        return t[i:] + go_type({**x, 'type': t[:i]}, prefix, structs)
    if t == 'tuple':
        fields=[]
        used=set()
        for i,c in enumerate(x['components']):
            name=exported(c.get('name') or f'Value{i}')
            name=unique_name(name,used,max(1,i))
            fields.append(f'{name} {go_type(c,prefix+name,structs)} `abi:"{c.get("name", "")}"`')
        register_struct(structs,prefix,'type '+prefix+' struct {\n'+'\n'.join(fields)+'\n}\n')
        return prefix
    if t=='address': return 'common.Address'
    if t in ('string','bool'): return t
    if t=='bytes': return '[]byte'
    if t=='function': return '[24]byte'
    if t.startswith('bytes'): return f'[{t[5:]}]byte'
    if t.startswith(('uint','int')):
        n=t.removeprefix('uint').removeprefix('int')
        return t if n in ('8','16','32','64') else '*big.Int'
    raise ValueError(t)

def generate(profile, entries):
    if not re.fullmatch(r'[A-Z][A-Za-z0-9_]*',profile): raise ValueError('invalid profile identifier')
    entries=normalize(entries)
    structs={}
    out=['// Code generated by tools/generate.py; DO NOT EDIT.\npackage decimal\n',
         'import("context";"math/big";"github.com/ethereum/go-ethereum/common";"github.com/ethereum/go-ethereum/accounts/abi")\n',
         'var _ = context.Background\nvar _ = big.NewInt\nvar _ common.Address\nvar _ = abi.ConvertType\n',
         f'type {profile} struct {{ *Contract }}\n',
         f'func (c *Client) {profile}(address common.Address) (*{profile},error) {{ contract,e:=c.Contract("{profile}",address);if e!=nil{{return nil,e}};return &{profile}{{contract}},nil }}\n']
    records=[]
    names=set()
    for m in sorted((x for x in entries if x.get('type')=='function'),key=signature):
        sig=signature(m); name=unique_name(exported(m['name']),names)
        prefix=profile+name
        args=[];ids=[]
        for i,x in enumerate(m['inputs']):
            arg=f'arg{i}';args.append(arg+' '+go_type(x,prefix+'Arg'+str(i),structs));ids.append(arg)
        argdecl=', '.join(args); argcall=(', '+', '.join(ids)) if ids else ''
        if m.get('stateMutability') in ('pure','view'):
            result=prefix+'Result';fields=[];assign=[]
            for i,x in enumerate(m['outputs']):
                typ=go_type(x,prefix+'Output'+str(i),structs)
                fields.append(f'Value{i} {typ}')
                assign.append(f'result.Value{i}=*abi.ConvertType(values[{i}],new({typ})).(*{typ})')
            register_struct(structs,result,f'type {result} struct {{ '+ '; '.join(fields)+' }\n')
            methodname='Read'+name
            out.append(f'// {methodname} calls {sig}.\nfunc (c *{profile}) {methodname}(ctx context.Context{", " if argdecl else ""}{argdecl}) ({result},error) {{\n'
                       f'var result {result}; values,e:=c.Read(ctx,"{sig}"{argcall});if e!=nil{{return result,e}};_ = values;\n'+ ';'.join(assign)+';return result,nil\n}\n')
        else:
            methodname='Build'+name
            value='value' if m.get('stateMutability')=='payable' else 'nil'
            params= ('value *big.Int'+(', ' if argdecl else '') if value=='value' else '')+argdecl
            out.append(f'// {methodname} builds {sig}; it does not sign or send.\nfunc (c *{profile}) {methodname}({params}) (Call,error) {{ return c.Build("{sig}",{value}{argcall}) }}\n')
        records.append({'profile':profile,'signature':sig,'method':methodname,'mutability':m.get('stateMutability'),'source_status':'pinned-abi'})
    (ROOT/f'{profile.lower()}_generated.go').write_text(''.join(out)+''.join(structs.values()),encoding='utf-8')
    return records

def main():
    p=argparse.ArgumentParser();[p.add_argument('--'+x,required=True,type=Path) for x in ('node','js','go')];args=p.parse_args()
    sources={'go-smart-node':args.node,'dsc-js-sdk':args.js,'dsc-go-sdk':args.go}
    for repo,path in sources.items():
        head=subprocess.check_output(['git','-C',str(path),'rev-parse','HEAD'],text=True).strip()
        if head!=PINS[repo]: raise SystemExit(f'{repo}: expected {PINS[repo]}, got {head}')
    (ROOT/'abi').mkdir(exist_ok=True); (ROOT/'docs').mkdir(exist_ok=True)
    inputs=[]
    for file in sorted((args.node/'contracts').rglob('*.json')):
        profile=file.stem.removeprefix('Decimal')
        inputs.append((profile,'go-smart-node',file))
    for name in ('Checks','Multicall'):
        inputs.append((name,'dsc-js-sdk',args.js/f'src/decimalevm/abi/{name}.json'))
    manifest=[];catalog=[]
    for profile,repo,file in inputs:
        rel=file.relative_to(sources[repo]).as_posix()
        # A pinned HEAD alone does not establish that a checked-out artifact is
        # unchanged. Refuse dirty imported files before assigning commit URLs.
        verify_source_file(sources[repo],rel)
        entries=normalize(json.loads(file.read_text(encoding='utf-8')))
        dest=ROOT/f'abi/{profile}.json';dest.write_text(json.dumps(entries,indent=2)+'\n',encoding='utf-8')
        rel=file.relative_to(sources[repo]).as_posix()
        manifest.append({'profile':profile,'repository':repo,'commit':PINS[repo],'path':rel,
                         'source_sha256':hashlib.sha256(file.read_bytes()).hexdigest(),
                         'abi_sha256':hashlib.sha256(dest.read_bytes()).hexdigest(),
                         'url':f'https://bitbucket.org/decimalteam/{repo}/src/{PINS[repo]}/{rel}'})
        catalog+=generate(profile,entries)
    (ROOT/'docs/sources.json').write_text(json.dumps({'pins':PINS,'abis':manifest},indent=2)+'\n')
    (ROOT/'docs/contract-catalog.json').write_text(json.dumps(catalog,indent=2)+'\n')
    api=['# Contract API\n\n', 'Generated typed argument/result wrappers use raw base units. `Build*` is offline; `Read*` calls RPC. '
         'Result fields `Value0`, `Value1`, etc. follow ABI output order. Tuple field names follow the ABI. '
         'Overloads receive deterministic numeric suffixes, ordered by the full signature.\n\n',
         '| Profile | Go method | Solidity signature | Mutability |\n|---|---|---|---|\n']
    for item in catalog:
        api.append(f'| {item["profile"]} | `{item["method"]}` | `{item["signature"]}` | {item["mutability"]} |\n')
    (ROOT/'docs/API.md').write_text(''.join(api),encoding='utf-8')
    # Inventory public JS methods and exported Go methods without executing upstream.
    inventories={}
    for repo,rel,pattern in [('dsc-js-sdk','src/decimalevm/index.ts',r'^  public async (\w+)\('),
                             ('dsc-js-sdk-call','src/decimalevm/call.ts',r'^    public async (\w+)\(')]:
        file=args.js/rel
        verify_source_file(args.js,rel)
        inventories[repo]=[{'name':m.group(1),'line':i,'path':rel} for i,line in enumerate(without_comments(file.read_text()).splitlines(),1) if (m:=re.search(pattern,line))]
    inventories['dsc-go-sdk']=[]
    for f in sorted((args.go/'decimalevm/methods').glob('*.go')):
        rel=f.relative_to(args.go).as_posix();verify_source_file(args.go,rel)
        inventories['dsc-go-sdk'] += [{'name':m.group(1),'path':rel,'line':i} for i,line in enumerate(f.read_text().splitlines(),1) if (m:=re.search(r'^func ([A-Z]\w*)\(',line))]
    (ROOT/'docs/upstream-inventory.json').write_text(json.dumps(inventories,indent=2)+'\n')
    (ROOT/'THIRD_PARTY_NODE_LICENSE').write_bytes((args.node/'LICENSE').read_bytes())
    (ROOT/'THIRD_PARTY_JS_LICENSE').write_text('Checks and Multicall ABI artifacts from dsc-js-sdk '+PINS['dsc-js-sdk']+'.\n'
        'The upstream package declares ISC (package.json); the repository snapshot contains no separate LICENSE file.\n'
        'ABI files are interfaces; no JS implementation is redistributed. Review attribution before publication.\n')
    print(f'{len(inputs)} ABI profiles, {len(catalog)} generated methods')

if __name__=='__main__': main()
