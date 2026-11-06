#!/usr/bin/env python3
"""Fetch immutable build inputs; validate every byte against dependencies.lock.json."""
import hashlib,json,os,pathlib,subprocess as sp,tarfile,sys,platform,socket,urllib.parse,zipfile
root=pathlib.Path(__file__).resolve().parents[1];os.chdir(root)
lock=json.loads((root/'dependencies.lock.json').read_text())['files'];(root/'.cache').mkdir(exist_ok=True)
def fetch(key):
 item=lock[key];p=root/item['path'];p.parent.mkdir(parents=True,exist_ok=True)
 if p.exists() and hashlib.sha256(p.read_bytes()).hexdigest()==item['sha256']:return p
 tmp=p.with_name(p.name+'.download')
 # Domestic mirror first; only verified identical bytes can be used. Official URL is fallback.
 sources=[item['url']]
 if 'github.com/' in item['url'] and '/releases/download/' in item['url']:sources.insert(0,'https://ghfast.top/'+item['url'])
 proxy=os.environ.get('https_proxy') or os.environ.get('HTTPS_PROXY')
 if proxy:
  u=urllib.parse.urlparse(proxy)
  try:
   with socket.create_connection((u.hostname,u.port or 8080),timeout=2):pass
  except OSError:raise SystemExit('代理未启用或不可达，请开启代理后重试；未修改现有依赖。')
 for url in sources:
  if proxy:print('Using configured proxy; ensure it is enabled.',file=sys.stderr)
  e=sp.run(['curl','-fL','--retry','1','--connect-timeout','10','--max-time','180',url,'-o',str(tmp)]).returncode
  if e==0 and hashlib.sha256(tmp.read_bytes()).hexdigest()==item['sha256']:tmp.replace(p);return p
 tmp.unlink(missing_ok=True);raise SystemExit(f'下载或 SHA-256 校验失败: {key}。未修改现有依赖。')
fetch('agh-linux.tar.gz')
if platform.system()=='Linux' and platform.machine() in ('x86_64','amd64'):fetch('agh-linux-amd64.tar.gz')
if platform.system()=='Darwin' and platform.machine()=='arm64':
 native=fetch('agh-darwin.zip');target=root/'.cache/darwin'
 with zipfile.ZipFile(native) as z:
  for name in z.namelist():
   if pathlib.PurePosixPath(name).is_absolute() or '..' in pathlib.PurePosixPath(name).parts:raise SystemExit('unsafe native test archive')
  z.extractall(target)
 (target/'AdGuardHome/AdGuardHome').chmod(0o755)
 p=fetch('go-toolchain');out=root/'.tools';out.mkdir(exist_ok=True)
 # Official verified Go archive; reject absolute paths and traversal before extraction.
 if not (out/'go/bin/go').exists():
  with tarfile.open(p) as tf:
   for m in tf.getmembers():
    if pathlib.PurePosixPath(m.name).is_absolute() or '..' in pathlib.PurePosixPath(m.name).parts:raise SystemExit('unsafe toolchain archive')
   tf.extractall(out)
print('固定内核/工具链已准备。规则、字体与 wrapper 为源码内固定输入；不会覆盖。')
