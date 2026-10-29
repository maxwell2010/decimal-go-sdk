"""Seed an isolated cache only with ZIPs matching this reference's go.sum Hash1.
Never changes shared cache. Run after go list -m -json all > modules.json.
"""
import base64, hashlib, json, pathlib, shutil, sys, zipfile
source=pathlib.Path(sys.argv[1]);dest=pathlib.Path(sys.argv[2]).resolve()
if dest==source.resolve() or 'node-validation' not in str(dest):raise SystemExit('isolated node-validation cache required')
sums={tuple(line.split()[:2]):line.split()[2] for line in pathlib.Path('go.sum').read_text().splitlines()}
text=pathlib.Path('modules.json').read_text(encoding='utf-8-sig');decoder=json.JSONDecoder();modules=[]
while text.strip():
    value,n=decoder.raw_decode(text.lstrip());modules.append(value);text=text.lstrip()[n:]
count=0;missing=[]
for module in modules:
    if module.get('Main'):continue
    module=module.get('Replace',module);path,version=module.get('Path'),module.get('Version')
    if not version:continue
    escaped=''.join('!'+c.lower() if c.isupper() else c for c in path)
    relative=pathlib.Path('cache/download')/escaped/'@v';cached=source/relative;zip_path=cached/(version+'.zip');expected=sums.get((path,version))
    if not zip_path.exists() or not expected:missing.append(path+'@'+version);continue
    target=dest/relative;target.mkdir(parents=True,exist_ok=True);copied=target/zip_path.name;shutil.copyfile(zip_path,copied)
    try:
        h=hashlib.sha256()
        with zipfile.ZipFile(copied) as archive:
            for name in sorted(archive.namelist()):
                if '\n' in name:raise ValueError('invalid ZIP filename')
                digest=hashlib.sha256(archive.read(name)).hexdigest();h.update((digest+'  '+name+'\n').encode())
        actual='h1:'+base64.b64encode(h.digest()).decode()
        if actual!=expected:raise ValueError('ZIP checksum mismatch')
    except Exception:
        copied.unlink();missing.append(path+'@'+version+' (rejected ZIP)');continue
    (target/(version+'.ziphash')).write_text(actual)
    for suffix in ('.mod','.info'):
        if (cached/(version+suffix)).exists():shutil.copyfile(cached/(version+suffix),target/(version+suffix))
    count+=1
print(json.dumps({'verified_zip_count':count,'unseeded':missing},indent=2))
