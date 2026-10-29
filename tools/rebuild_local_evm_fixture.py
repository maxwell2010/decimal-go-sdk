"""Rebuild historical official NewToken using the pinned solc npm lock.

Create a disposable compiler directory, copy testdata/evm/compiler-package*.json
to package.json/package-lock.json, run npm ci --ignore-scripts --no-audit --no-fund.
Pass its absolute node_modules/solc directory. The source never uses imports.
"""
import argparse
import hashlib
from pathlib import Path
import subprocess
ROOT=Path(__file__).resolve().parents[1]
CODE="const fs=require('fs');const solc=require(process.argv[1]);if(solc.version()!=='0.8.17+commit.8df45f5f.Emscripten.clang')throw new Error('Pinned solc 0.8.17 required');const source=fs.readFileSync(process.argv[2],'utf8');const input={language:'Solidity',sources:{'decimalToken.sol':{content:source}},settings:{optimizer:{enabled:false,runs:200},evmVersion:'london',outputSelection:{'*':{'*':['abi','evm.bytecode.object','evm.deployedBytecode.object']}}}};const r=JSON.parse(solc.compile(JSON.stringify(input)));if((r.errors||[]).some(e=>e.severity==='error'))throw new Error(JSON.stringify(r.errors));const c=r.contracts['decimalToken.sol'].NewToken;process.stdout.write(JSON.stringify({compiler:solc.version(),contractName:'NewToken',abi:c.abi,bytecode:'0x'+c.evm.bytecode.object,deployedBytecode:'0x'+c.evm.deployedBytecode.object,settings:input.settings},null,2)+'\\n');"
def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('solc');p.add_argument('--check',action='store_true');a=p.parse_args()
 if hashlib.sha256((ROOT/'testdata/evm/decimalToken.sol').read_bytes()).hexdigest()!='686b1b0fb9f45928f8ed5ebf416b77b6e1ea9ccfa27d106d5acdbb277230e3e3':raise SystemExit('Official source SHA-256 mismatch')
 r=subprocess.run(['node','-e',CODE,str(Path(a.solc).resolve()),str(ROOT/'testdata/evm/decimalToken.sol')],capture_output=True,check=True)
 target=ROOT/'testdata/evm/NewToken.compiled.json'
 if a.check:
  if r.stdout!=target.read_bytes():raise SystemExit('Compiler output differs')
  print('Pinned NewToken compiler output matches')
 else:target.write_bytes(r.stdout)
if __name__=='__main__':main()
