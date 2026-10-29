"""Rebuild the official OpenZeppelin v4.7.2 standard permit fixture.
Pass the pinned solc 0.8.17 module directory used by rebuild_local_evm_fixture.py.
This upstream mock exercises EIP2612 only, not a Decimal deployed contract.
"""
import argparse,hashlib,json,subprocess
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
CODE="const fs=require('fs'),solc=require(process.argv[1]);if(solc.version()!=='0.8.17+commit.8df45f5f.Emscripten.clang')throw new Error('Pinned solc required');const input=JSON.parse(fs.readFileSync(process.argv[2],'utf8'));const r=JSON.parse(solc.compile(JSON.stringify(input)));if((r.errors||[]).some(e=>e.severity==='error'))throw new Error(JSON.stringify(r.errors));const c=r.contracts['contracts/mocks/ERC20PermitMock.sol'].ERC20PermitMock;process.stdout.write(JSON.stringify({compiler:solc.version(),abi:c.abi,bytecode:'0x'+c.evm.bytecode.object,deployedBytecode:'0x'+c.evm.deployedBytecode.object},null,2)+'\\n');"
def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('solc');p.add_argument('--check',action='store_true');a=p.parse_args()
 source=ROOT/'testdata/evm/permit-input.json'
 if hashlib.sha256(source.read_bytes()).hexdigest()!='46591f61c9000a86c291a2019854657bb585f65ff74bb61db3f1e0f01c8a9081':raise SystemExit('Pinned official input mismatch')
 r=subprocess.run(['node','-e',CODE,str(Path(a.solc).resolve()),str(source)],capture_output=True,check=True)
 target=ROOT/'testdata/evm/permit-compiled.json'
 if a.check:
  if target.read_bytes()!=r.stdout:raise SystemExit('Permit compiler output differs')
  print('Pinned standard EIP2612 fixture matches')
 else:target.write_bytes(r.stdout)
if __name__=='__main__':main()
