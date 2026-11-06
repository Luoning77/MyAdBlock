#!/usr/bin/env python3
"""Create a public source snapshot from explicit source paths, excluding reference/cache/private."""
from pathlib import Path
import gzip,hashlib,io,json,os,tarfile
root=Path(__file__).resolve().parents[1];os.chdir(root)
v=dict(l.split('=',1) for l in Path('version.properties').read_text().splitlines() if '=' in l)
paths={}
roots=['android','cmd','internal','module','presets','tools','tests','licenses','docs','webui/src','webui/gradle','webui/webpack.config.d','webui/kotlin-js-store','.github']
for base in roots:
 for p in Path(base).rglob('*'):
  if p.is_symlink():raise SystemExit('Refusing source symlink: '+str(p))
  if p.is_file() and '__pycache__' not in p.parts:paths[str(p)]=p
for n in ['.gitignore','README.md','README_EN.md','PRODUCT.md','DESIGN.md','LICENSE','NOTICE.md','version.properties','dependencies.lock.json','go.mod','go.sum','webui/build.gradle.kts','webui/settings.gradle.kts','webui/gradle.properties','webui/gradlew','webui/go.mod','.impeccable/design.json']:paths[n]=Path(n)
manifest={'versions':v,'files':{n:hashlib.sha256(p.read_bytes()).hexdigest() for n,p in sorted(paths.items())}}
out=Path('dist')/('MyAdBlock-'+v['moduleVersion']+'-source.tar.gz');out.parent.mkdir(exist_ok=True)
with out.open('wb') as raw,gzip.GzipFile(filename='',mode='wb',fileobj=raw,mtime=0,compresslevel=9) as gz,tarfile.open(fileobj=gz,mode='w') as tf:
 for n,p in sorted(paths.items()):
  b=p.read_bytes();info=tarfile.TarInfo('MyAdBlock/'+n);info.size=len(b);info.mtime=1767225600;info.uid=info.gid=0;info.uname=info.gname='root';info.mode=0o755 if p.stat().st_mode&0o111 else 0o644;tf.addfile(info,io.BytesIO(b))
 b=(json.dumps(manifest,ensure_ascii=False,sort_keys=True,indent=2)+'\n').encode();info=tarfile.TarInfo('MyAdBlock/SOURCE_MANIFEST.json');info.size=len(b);info.mtime=1767225600;info.mode=0o644;tf.addfile(info,io.BytesIO(b))
with tarfile.open(out) as tf:
 names=tf.getnames();assert not any(any(x in Path(n).parts for x in ('reference','private','.cache','.tools','node_modules')) or n.endswith('/prompt.md') for n in names)
 for n,h in manifest['files'].items():assert hashlib.sha256(tf.extractfile('MyAdBlock/'+n).read()).hexdigest()==h
sha=hashlib.sha256(out.read_bytes()).hexdigest();out.with_suffix('.gz.sha256').write_text(sha+'  '+out.name+'\n');print(out);print(sha)
