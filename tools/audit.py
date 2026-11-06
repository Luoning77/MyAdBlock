#!/usr/bin/env python3
"""Public source/package invariants; never scans the private reference tree."""
from pathlib import Path
import re,sys,zipfile,struct,json,hashlib
ROOT=Path(__file__).resolve().parents[1]
for p in list((ROOT/'module').rglob('*.sh'))+list((ROOT/'internal').rglob('*.go')):
 if p.name.endswith('_test.go'): continue
 s=p.read_text()
 for bad in ('-F OUTPUT','-F PREROUTING','--dport 853 -j DROP','airplane_mode_on','chattr ','/data/system/ifw','/data/data/com.'):
  assert bad not in s,(p,bad)
assert 'prompt.md' in (ROOT/'.gitignore').read_text()
for f in (ROOT/'webui/build/dist/js/productionExecutable').glob('*.js'):
 assert not re.search(r'file:///(Users|home|private)/',f.read_text()),f'Build machine path in {f}'
assert 'root' in (ROOT/'internal/config/config.go').read_text()
for p in (ROOT/'module').glob('*.sh'):assert '\r' not in p.read_text()
if len(sys.argv)>1:
 with zipfile.ZipFile(sys.argv[1]) as z:
  files=set(z.namelist())
  required={'module.prop','service.sh','customize.sh','action.sh','uninstall.sh','emergency.sh','bin/myadblock','bin/AdGuardHome','webroot/myadblock.js','webroot/skiko.wasm','lib/network-probe.jar','presets/snapshots/1.txt','presets/snapshots/2.txt','presets/snapshots/3.txt','LICENSE','NOTICE.md','version.properties','META-INF/com/google/android/update-binary'}
  assert required<=files,required-files
  assert not any(n.startswith(('reference/','private/','.git/')) or n.endswith(('.map','.secret')) or n=='prompt.md' for n in files)
  for name in ('bin/AdGuardHome','bin/myadblock'):
   b=z.read(name);assert b[:4]==b'\x7fELF' and b[4]==2 and struct.unpack_from('<H',b,18)[0]==183,name
  assert '127.0.0.1' in z.read('action.sh').decode() or 'grant' in z.read('action.sh').decode()
  manifest=json.loads(z.read('MANIFEST.json'))
  for n,h in manifest['files'].items():assert hashlib.sha256(z.read(n)).hexdigest()==h,n
print('Static safety / package audit passed')
