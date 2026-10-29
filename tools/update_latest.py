"""Advance the public install channel only after its GitHub release is published."""
import argparse
import json
import os
from pathlib import Path
import re
import urllib.request

ROOT = Path(__file__).resolve().parents[1]
MODULE = 'github.com/maxwell2010/decimal-go-sdk'
REPOSITORY = 'https://github.com/maxwell2010/decimal-go-sdk'


def release_manifest(version, release):
    if not re.fullmatch(r'v[01]\.(?:0|[1-9][0-9]*)\.(?:0|[1-9][0-9]*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?', version):
        raise ValueError('invalid or unsupported public module version')
    if '-' in version:
        for part in version.split('-', 1)[1].split('.'):
            if part.isascii() and part.isdigit() and len(part) > 1 and part.startswith('0'):
                raise ValueError('noncanonical prerelease version')
    page = REPOSITORY + '/releases/tag/' + version
    if release.get('draft') is not False or release.get('tag_name') != version or release.get('html_url') != page:
        raise ValueError('expected a published release at the exact version')
    stem = 'decimal-go-sdk-' + version[1:]
    base = REPOSITORY + '/releases/download/' + version + '/'
    names = [stem + '.zip', stem + '.tar.gz', 'SHA256SUMS']
    for name in names:
        matches = [a for a in release.get('assets', []) if a.get('name') == name]
        if len(matches) != 1 or matches[0].get('state') != 'uploaded' or matches[0].get('size', 0) <= 0 or matches[0].get('browser_download_url') != base + name:
            raise ValueError('published release asset missing or inconsistent: ' + name)
    return {'schema': 1, 'version': version, 'go': '1.26.10',
            'evm': MODULE + '@' + version, 'cosmos': MODULE + '/cosmos@' + version,
            'release_url': page, 'source_zip': base + names[0],
            'source_tar_gz': base + names[1], 'checksums': base + names[2]}


class NoRedirects(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, *args, **kwargs):
        return None


def published_release(version):
    # Validate the version before constructing a remote URL.
    if not re.fullmatch(r'v[01]\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?', version):
        raise ValueError('invalid release version')
    request = urllib.request.Request('https://api.github.com/repos/maxwell2010/decimal-go-sdk/releases/tags/' + version,
                                     headers={'User-Agent': 'decimal-go-sdk-release-channel'})
    with urllib.request.build_opener(NoRedirects).open(request, timeout=20) as response:
        raw = response.read((1 << 20) + 1)
    if len(raw) > 1 << 20:
        raise ValueError('release response exceeds 1 MiB')
    return json.loads(raw)


def update(version, release, destination):
    data = release_manifest(version, release)
    temporary = destination.with_suffix('.json.tmp')
    try:
        temporary.write_text(json.dumps(data, indent=2) + '\n', encoding='utf-8', newline='\n')
        os.replace(temporary, destination)
    finally:
        temporary.unlink(missing_ok=True)
    return data


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--version', default=(ROOT / 'VERSION').read_text().strip())
    args = parser.parse_args()
    data = update(args.version, published_release(args.version), ROOT / 'latest.json')
    print('Published install channel:', data['version'])


if __name__ == '__main__':
    main()
