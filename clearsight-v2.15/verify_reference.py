#!/usr/bin/env python3
"""Verify an untouched ClearSight design release; no network or dependencies."""
from __future__ import annotations
import hashlib,json,sys
from pathlib import Path,PurePosixPath
ROOT=Path(__file__).resolve().parent

def verify(root:Path=ROOT)->int:
 manifest=json.loads((root/'MANIFEST.json').read_text(encoding='utf-8'))
 if manifest.get('release')!='2.15':raise ValueError('Unexpected reference release')
 expected={}
 for item in manifest['files']:
  name=item['path'];parts=PurePosixPath(name).parts
  if not parts or name.startswith('/') or any(p in ('.','..') for p in parts) or '\\' in name:raise ValueError('Unsafe manifest path')
  if name in expected:raise ValueError('Duplicate manifest path: '+name)
  expected[name]=item
 actual={p.relative_to(root).as_posix() for p in root.rglob('*') if p.is_file() and '__pycache__' not in p.parts and p.name!='MANIFEST.json'}
 if actual!=expected.keys():raise ValueError(f'Manifest drift: missing={sorted(expected.keys()-actual)} extra={sorted(actual-expected.keys())}')
 for name,item in expected.items():
  path=root/name
  if path.is_symlink():raise ValueError('Symlink not allowed: '+name)
  data=path.read_bytes()
  if len(data)!=item['bytes'] or hashlib.sha256(data).hexdigest()!=item['sha256']:raise ValueError('Changed reference file: '+name)
 print(f'ClearSight V2.15 reference integrity verified: {len(expected)} files.')
 return len(expected)
if __name__=='__main__':
 try:verify()
 except (OSError,ValueError,TypeError,KeyError) as exc:print('Reference integrity failure:',exc,file=sys.stderr);sys.exit(1)
