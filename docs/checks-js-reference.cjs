const fs = require('fs');
// Offline public fixtures only. Pass an installed, lock-verified ethers5.7.2 directory.
const { ethers } = require(process.argv[2]);
(async () => {
  if (ethers.version !== 'ethers/5.7.2') throw Error('Expected ethers/5.7.2');
  const vectors = [];
  for (const password of ['decimal-check-public-fixture', '\u041f\u0440\u043e\u0432\u0435\u0440\u043a\u0430-\u2713', 'nul\0fixture']) {
    const private_key = ethers.utils.id(password);
    const wallet = new ethers.Wallet(private_key);
    const check_hash = '0x' + '01'.repeat(32);
    const redeemer = '0x2000000000000000000000000000000000000002';
    const packed = ethers.utils.solidityPack(['bytes32', 'address'], [check_hash, redeemer]);
    const message_hash = ethers.utils.solidityKeccak256(['bytes32', 'address'], [check_hash, redeemer]);
    const personal_digest = ethers.utils.hashMessage(ethers.utils.arrayify(message_hash));
    const signature = await wallet.signMessage(ethers.utils.arrayify(message_hash));
    const { v, r, s } = ethers.utils.splitSignature(signature);
    const recovered = ethers.utils.verifyMessage(ethers.utils.arrayify(message_hash), signature);
    vectors.push({password, private_key, signer: wallet.address, check_hash, redeemer, packed, message_hash, personal_digest, signature, v, r, s, recovered});
  }
  console.log(JSON.stringify({fixture_only: true, implementation: 'ethers/5.7.2', algorithm: 'pinned official JS index.ts:1862-1900; current index.ts:1988-2026', vectors, empty_password_upstream: 'rejected before ethers.utils.id'}, null, 2));
})().catch(e => { console.error(e.message); process.exitCode=1; });
