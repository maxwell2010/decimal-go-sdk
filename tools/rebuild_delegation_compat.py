"""Reproduce the recovery ABI from the exact pinned official Go-binding bytes.

python -B tools/rebuild_delegation_compat.py path/to/DecimalDelegation.go --check
This tool reads a local source file. It never executes Go or downloads an ABI.
"""
import argparse
import hashlib
import json
from pathlib import Path
import re

ROOT = Path(__file__).resolve().parents[1]
COMMIT = 'a46cb48d7965c1a64a495cebb22e735dfe4b9da3'
SOURCE_SHA256 = '248199aadf86bb4bd6836b70f13f0784b976e7edf65e7908f2c34e6bebd714bb'
ABI_SHA256 = '600ea8ed2c542b284e7eadf222e738c13663f029becf22b056b56678c5948fa0'
DESTINATION = ROOT / 'delegation_compat/Delegation20261030.json'


def render_binding(raw):
    literals = re.findall(r'\bABI:\s*("(?:[^"\\]|\\.)*")\s*,', raw.decode('utf-8'))
    if len(literals) != 1:
        raise ValueError('expected exactly one literal ABI in official Go-binding')
    entries = json.loads(json.loads(literals[0]))
    if not isinstance(entries, list) or not all(isinstance(x, dict) for x in entries):
        raise ValueError('binding ABI must be an array of entries')
    return (json.dumps(entries, separators=(',', ':')) + '\n').encode('utf-8')


def rebuild(source, destination=DESTINATION, check=False):
    raw = source.read_bytes()
    if hashlib.sha256(raw).hexdigest() != SOURCE_SHA256:
        raise ValueError('source does not match pinned official Go-binding SHA256')
    rendered = render_binding(raw)
    if hashlib.sha256(rendered).hexdigest() != ABI_SHA256:
        raise ValueError('extracted ABI does not match the reviewed pinned ABI SHA256')
    if check:
        if destination.read_bytes() != rendered:
            raise ValueError('checked-in compatibility ABI differs from official Go-binding')
    else:
        destination.write_bytes(rendered)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('source', type=Path)
    parser.add_argument('--check', action='store_true')
    args = parser.parse_args()
    try:
        rebuild(args.source, check=args.check)
    except (ValueError, OSError) as exc:
        parser.exit(1, f'ABI reproduction refused: {exc}\n')
    print('Official delegation ABI reproduced:', COMMIT, ABI_SHA256)


if __name__ == '__main__':
    main()
