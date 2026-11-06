#!/usr/bin/env python3
"""Build static ARM64 controller, DEX helper, offline resources and deterministic module ZIP."""
from pathlib import Path
import subprocess as sp,os,shutil,zipfile,hashlib,json,argparse,struct,tarfile
ROOT=Path(__file__).resolve().parents[1];os.chdir(ROOT)
p=argparse.ArgumentParser();p.add_argument('--skip-webui',action='store_true');p.add_argument('--private-config',type=Path);a=p.parse_args()
versions=dict(l.strip().split('=',1) for l in (ROOT/'version.properties').read_text().splitlines() if '=' in l)
ver_year,ver_major,ver_build=map(int,versions['moduleVersion'].split('.'))
assert 0<=ver_year<=99 and 1<=ver_major<=99 and 1<=ver_build<=9999,'版本超出 Root 管理器 versionCode 派生范围'
versions['moduleVersionCode']=str(ver_year*1000000+ver_major*10000+ver_build)
cache=ROOT/'.cache';build=ROOT/'build';stage=build/'module';stage.mkdir(parents=True,exist_ok=True)
lock=json.loads((ROOT/'dependencies.lock.json').read_text())
assert versions['aghVersion']==lock['files']['agh-linux.tar.gz']['version'].lstrip('v'),'aghVersion 与锁定内核不一致；请同步校验依赖后构建'
def verify(path,key):
 assert hashlib.sha256(path.read_bytes()).hexdigest()==lock['files'][key]['sha256'],f'依赖校验失败 {key}'
verify(cache/'agh-linux.tar.gz','agh-linux.tar.gz')
verify(ROOT/lock['files']['misans-font']['path'],'misans-font')
for i in range(1,4):verify(ROOT/f'presets/snapshots/{i}.txt',f'snapshot-{i}')
go=os.environ.get('GO',str(ROOT/'.tools/go/bin/go'))
env=os.environ.copy();env.update(CGO_ENABLED='0',GOOS='linux',GOARCH='arm64',GOMODCACHE=str(cache/'gomod'),GOCACHE=str(cache/'gobuild'))
env.setdefault('GOPROXY','https://goproxy.cn,https://proxy.golang.org,direct')
sp.run([go,'build','-mod=readonly','-trimpath','-buildvcs=false','-ldflags=-s -w -buildid=','-o',str(build/'myadblock'),'./cmd/myadblock'],env=env,check=True)
if not a.skip_webui:sp.run(['webui/gradlew','--no-daemon','jsBrowserDistribution'],check=True)
web=ROOT/'webui/build/dist/js/productionExecutable';assert (web/'myadblock.js').is_file(),'先构建完整 Miuix WebUI'
web_package=json.loads((ROOT/'webui/build/js/packages/MyAdBlockWebUI/package.json').read_text())
assert web_package['version']==versions['webuiVersion'],'WebUI 版本已变化，请先重新构建前端'
sdk=Path(os.environ.get('ANDROID_HOME',str(Path.home()/'Library/Android/sdk')));platform=sdk/'platforms/android-37.0/android.jar';d8=sdk/'build-tools/36.0.0/d8'
classes=build/'java';classes.mkdir(exist_ok=True);dex=build/'dex';dex.mkdir(exist_ok=True)
java17=Path('/opt/homebrew/opt/openjdk@17/bin');javac=str(java17/'javac') if (java17/'javac').exists() else 'javac'
sp.run([javac,'--release','8','-classpath',str(platform),'-d',str(classes),'android/NetworkProbe.java'],check=True)
sp.run([str(d8),'--min-api','29','--lib',str(platform),'--output',str(dex),*map(str,classes.rglob('*.class'))],check=True)
if stage.exists():shutil.rmtree(stage)
shutil.copytree('module',stage);(stage/'bin').mkdir();(stage/'lib').mkdir()
shutil.copy2(build/'myadblock',stage/'bin/myadblock');
with tarfile.open(cache/'agh-linux.tar.gz','r:gz') as tf:(stage/'bin/AdGuardHome').write_bytes(tf.extractfile(next(m for m in tf.getmembers() if m.name.lstrip('./')=='AdGuardHome/AdGuardHome')).read())
with zipfile.ZipFile(stage/'lib/network-probe.jar','w',zipfile.ZIP_DEFLATED) as z:
 zi=zipfile.ZipInfo('classes.dex',(2026,1,1,0,0,0));zi.compress_type=zipfile.ZIP_DEFLATED;z.writestr(zi,(dex/'classes.dex').read_bytes())
shutil.copytree(web,stage/'webroot',ignore=shutil.ignore_patterns('*.map'))
shutil.copytree('presets',stage/'presets');shutil.copytree('licenses',stage/'licenses')
for n in ['LICENSE','NOTICE.md','version.properties','dependencies.lock.json']:shutil.copy2(n,stage/n)
shutil.copytree('docs',stage/'docs')
(stage/'audit').mkdir()
for f in ['go.mod','go.sum','webui/gradle/verification-metadata.xml','webui/kotlin-js-store/package-lock.json']:
 shutil.copy2(f,stage/'audit'/Path(f).name)
(stage/'module.prop').write_text(f'id=myadblock\nname=MyAdBlock\nversion={versions["moduleVersion"]}\nversionCode={versions["moduleVersionCode"]}\nauthor=Luoning77\ndescription=ARM64 DNS 过滤；安全回退、Miuix 管理；默认 root/root，请修改密码。实机兼容待验证。\n')
(stage/'skip_mount').touch()
if a.private_config:
 # Only a portable YAML subset is accepted on-device; offline first preset import is explicit and isolated.
 assert a.private_config.resolve().is_relative_to((ROOT/'private').resolve()),'个人配置必须放在已忽略的 private/ 目录'
 assert a.private_config.stat().st_size<=2<<20
 shutil.copy2(a.private_config,stage/'presets/personal.yaml')
manifest={'versions':versions,'variant':'personal' if a.private_config else 'public','files':{p.relative_to(stage).as_posix():hashlib.sha256(p.read_bytes()).hexdigest() for p in sorted(stage.rglob('*')) if p.is_file()}}
(stage/'MANIFEST.json').write_text(json.dumps(manifest,ensure_ascii=False,sort_keys=True,indent=2)+'\n')
dist=ROOT/'dist';dist.mkdir(exist_ok=True);suffix='-personal' if a.private_config else '';target=dist/f'MyAdBlock-{versions["moduleVersion"]}-arm64{suffix}.zip'
with zipfile.ZipFile(target,'w',zipfile.ZIP_DEFLATED,compresslevel=9) as z:
 for f in sorted(stage.rglob('*')):
  if not f.is_file():continue
  rel=f.relative_to(stage).as_posix();zi=zipfile.ZipInfo(rel,(2026,1,1,0,0,0));zi.compress_type=zipfile.ZIP_DEFLATED;zi.create_system=3;zi.external_attr=(0o100755 if rel.startswith('bin/') or rel.endswith('.sh') or rel.endswith('update-binary') else 0o100644)<<16;z.writestr(zi,f.read_bytes())
sha=hashlib.sha256(target.read_bytes()).hexdigest();target.with_suffix('.zip.sha256').write_text(sha+'  '+target.name+'\n')
sp.run(['python3','tools/audit.py',str(target)],check=True)
print(target);print(sha)
