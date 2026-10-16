#!/usr/bin/env python3
"""sources.py [--check]: writes the offline sources the Flatpak build reads (Flathub builds without network).

go-sources.json: every Go module file (.mod, .zip; .info is not byte-stable on the proxy) the wasm build (root module, GOOS=js) and the desktop build (desktop/, with its
replace) fetch, as a GOPROXY=file:// tree under goproxy/. Found by filling a scratch GOMODCACHE from proxy.golang.org with
`go list -deps`, so it needs network and Go; the files are hashed as downloaded.

node-sources.json: pnpm itself (package.json's packageManager) and every package in frontend/pnpm-lock.yaml, as a
static registry tree under npm-registry/ (<name>/-/<file>.tgz) that build.sh serves on loopback. Packages for another
OS or for musl are left out; ones for one CPU are limited to that arch.

--check regenerates into memory and fails when either file differs.
"""
import base64
import hashlib
import json
import os
import re
import subprocess
import sys
import tempfile
from pathlib import Path

here = Path(__file__).resolve().parent
repo = here.parents[2]
registry = 'https://registry.npmjs.org/'
proxy = 'https://proxy.golang.org/'
arches = {'x64': 'x86_64', 'arm64': 'aarch64'}

def go_sources():
    with tempfile.TemporaryDirectory() as cache:
        env = dict(os.environ, GOMODCACHE=cache, GOFLAGS='-mod=readonly', GOPROXY=proxy.rstrip('/'), GOTOOLCHAIN='local')
        subprocess.run(['go', 'list', '-deps', './wasm'], cwd=repo, env=dict(env, GOOS='js', GOARCH='wasm'), check=True, stdout=subprocess.DEVNULL)
        subprocess.run(['go', 'list', '-deps', '-tags', 'gtk3,production', '.'], cwd=repo / 'desktop', env=dict(env, GOOS='linux', CGO_ENABLED='1'), check=True, stdout=subprocess.DEVNULL)
        root = Path(cache, 'cache', 'download')
        out = []
        for f in sorted(root.rglob('*')):
            if f.suffix not in ('.mod', '.zip') or f.parent.name != '@v':
                continue
            rel = f.relative_to(root).as_posix()
            out.append({
                'type': 'file',
                'url': proxy + rel,
                'sha256': sha256(f),
                'dest': 'goproxy/' + str(Path(rel).parent.as_posix()),
                'dest-filename': f.name,
            })
        subprocess.run(['go', 'clean', '-modcache'], env=env, check=True)
        return out

def sha256(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()

def lock_packages(text):
    """The packages: section of a v9 pnpm-lock.yaml: {'name@version': {'integrity':…, 'cpu':[…], 'os':[…], 'libc':[…]}}."""
    pkgs, cur, inside = {}, None, False
    for line in text.splitlines():
        if not line.strip():
            continue
        if not line.startswith(' '):
            inside = line.rstrip() == 'packages:'
            continue
        if not inside:
            continue
        if line.startswith('  ') and not line.startswith('   '):
            cur = pkgs.setdefault(line.strip().rstrip(':').strip("'"), {})
        elif cur is not None and line.startswith('    ') and not line.startswith('     '):
            key, _, val = line.strip().partition(': ')
            if key == 'resolution':
                m = re.search(r'integrity: ([^,}]+)', val)
                if not m:
                    raise SystemExit(f'sources: {line.strip()}: only registry tarballs with an integrity are supported')
                cur['integrity'] = m.group(1).strip()
            elif key in ('cpu', 'os', 'libc'):
                cur[key] = [v.strip().strip("'") for v in val.strip('[]').split(',')]
    return pkgs

def tarball(name, version, integrity):
    algo, _, b64 = integrity.partition('-')
    if algo != 'sha512':
        raise SystemExit(f'sources: {name}@{version}: integrity {algo}, want sha512')
    base = name.rsplit('/', 1)[-1]
    return {
        'type': 'file',
        'url': f'{registry}{name}/-/{base}-{version}.tgz',
        'sha512': base64.b64decode(b64).hex(),
        'dest': f'npm-registry/{name}/-',
        'dest-filename': f'{base}-{version}.tgz',
    }

def node_sources():
    pkg = json.loads((repo / 'frontend' / 'package.json').read_text())
    pm, _, digest = pkg['packageManager'].partition('+sha512.')
    name, _, version = pm.rpartition('@')
    out = [dict(tarball(name, version, 'sha512-' + base64.b64encode(bytes.fromhex(digest)).decode()), dest='pnpm-dist')]
    for key, meta in sorted(lock_packages((repo / 'frontend' / 'pnpm-lock.yaml').read_text()).items()):
        if 'os' in meta and 'linux' not in meta['os']:
            continue
        if 'libc' in meta and 'glibc' not in meta['libc']:
            continue
        src = tarball(*key.rsplit('@', 1), meta['integrity'])
        if 'cpu' in meta:
            only = sorted(arches[c] for c in meta['cpu'] if c in arches)
            if not only:
                continue
            src['only-arches'] = only
        out.append(src)
    return out

def main():
    files = {'go-sources.json': go_sources(), 'node-sources.json': node_sources()}
    stale = []
    for name, data in files.items():
        text = json.dumps(data, indent=2) + '\n'
        path = here / name
        if '--check' in sys.argv:
            if not path.exists() or path.read_text() != text:
                stale.append(name)
        else:
            path.write_text(text)
            print(f'sources: {name}: {len(data)} files')
    if stale:
        raise SystemExit(f'sources: stale {", ".join(stale)}: run `task flatpak-sources`')

if __name__ == '__main__':
    main()
