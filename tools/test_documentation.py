"""Offline regression check for repository Markdown file links.

Checks file existence, not remote URLs or heading anchors. Fenced examples are
not documentation navigation. Dependency caches are pruned before traversal.
"""
import os
from pathlib import Path
import re
import tempfile
import unittest
from urllib.parse import unquote, urlsplit

ROOT = Path(__file__).resolve().parents[1]
SKIP = {'.tools', '.git', 'dist', 'vendor', 'node_modules', '__pycache__'}


def markdown_files(root):
    for parent, directories, files in os.walk(root):
        directories[:] = sorted(x for x in directories if x not in SKIP)
        for filename in sorted(files):
            if filename.lower().endswith('.md'):
                yield Path(parent) / filename


def local_destinations(text):
    # Preserve inline prose but ignore fenced source examples.
    text = re.sub(r'^\s*(`{3,}|~{3,})[^\n]*\n.*?^\s*\1\s*$', '', text,
                  flags=re.M | re.S)
    destinations = re.findall(r'!?\[[^\]\n]*\]\((<[^>]*>|[^\s)]+)(?:\s+"[^"\n]*")?\)', text)
    destinations += re.findall(r'^\s*\[[^\]\n]+\]:\s*(<[^>]*>|\S+)', text, flags=re.M)
    for destination in destinations:
        destination = destination.strip('<>')
        parsed = urlsplit(destination)
        if parsed.scheme or parsed.netloc or not parsed.path or parsed.path.startswith('/'):
            continue
        # Documentation templates sometimes name a literal placeholder path.
        if any(value in parsed.path for value in ('{', '}', '<', '>')):
            continue
        yield unquote(parsed.path)


def broken_links(root):
    broken = []
    checked = 0
    for source in markdown_files(root):
        for destination in local_destinations(source.read_text(encoding='utf-8-sig')):
            checked += 1
            if not (source.parent / destination).exists():
                broken.append(f'{source.relative_to(root)} -> {destination}')
    return checked, broken


class DocumentationLinksTest(unittest.TestCase):
    def test_russian_guides_have_no_utf8_cp1251_corruption(self):
        # UTF-8 decoded as CP1251 turns common Cyrillic letters into these pairs.
        markers = ['\u0420\u00b0', '\u0420\u00b5', '\u0420\u0451', '\u0421\u201a']
        for source in markdown_files(ROOT):
            if source.name.endswith('.ru.md'):
                text = source.read_text(encoding='utf-8-sig')
                self.assertLess(sum(text.count(marker) for marker in markers), 3,
                                'Corrupted Cyrillic in ' + str(source.relative_to(ROOT)))

    def test_repository_local_markdown_links_exist(self):
        count, broken = broken_links(ROOT)
        self.assertGreater(count, 0, 'No Markdown links checked')
        self.assertEqual(broken, [], '\n'.join(broken))

    def test_only_local_navigation_is_checked(self):
        text = '''[web](https://example.com/missing) [mail](mailto:hi@example.com)
[heading](#section) [file](some%20file.md?raw=1#section) [absolute](/path/to/file.md) [windows](C:/source/file.md)
[placeholder]({path}/example.md) [quoted](<space file.md> "Title")
[reference]: reference.md
```go
// [sample](not-a-link.md)
```
'''
        self.assertEqual(list(local_destinations(text)),
                         ['some file.md', 'space file.md', 'reference.md'])

    def test_missing_link_is_reported_and_cache_is_pruned(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            (root/'good.md').write_text('[missing](missing.md)', encoding='utf-8')
            (root/'.tools').mkdir()
            (root/'.tools'/'dependency.md').write_text('[ignored](other.md)', encoding='utf-8')
            count, broken = broken_links(root)
            self.assertEqual(count, 1)
            self.assertEqual(broken, ['good.md -> missing.md'])


if __name__ == '__main__':
    unittest.main()
