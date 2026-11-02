"""Verify external module archives without treating local source as an archive.

The independent reference implementation is a local test oracle. Its files, and the adapted IAVL/ICS23
sources, are checked against environment.json by audit_results.py.
"""
from pathlib import Path
import json
import re
import subprocess
import tempfile

ROOT = Path(__file__).resolve().parent

def main():
    with tempfile.TemporaryDirectory(prefix='ads-dependencies-') as directory:
        target = Path(directory) / 'verify.mod'
        source = (ROOT / 'go.mod').read_text()
        source = re.sub(r'^\s*data-ownership-system v[^\n]+\n', '', source, flags=re.M)
        source = re.sub(r'^replace data-ownership-system =>[^\n]+\n', '', source, flags=re.M)
        source = re.sub(
            r'^(replace \S+ => )\./([^\n]+)$',
            lambda match: match[1] + json.dumps(str(ROOT / match[2]), ensure_ascii=False),
            source, flags=re.M)
        target.write_text(source)
        target.with_suffix('.sum').write_bytes((ROOT / 'go.sum').read_bytes())
        print('External module archives; local sources are checked by SHA-256 and correctness tests.', flush=True)
        subprocess.run(['go', 'mod', 'verify', '-modfile=' + str(target)], cwd=ROOT, check=True)

if __name__ == '__main__':
    main()
