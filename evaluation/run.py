"""Run the HT-Merkle scalability, categorization and real-dataset experiments."""
from pathlib import Path
import hashlib,json,subprocess,os,platform,time
R=Path(__file__).resolve().parent;C=R.parent/'comparison';O=R/'results'
for d in ['logs','jobs']:(O/d).mkdir(parents=True,exist_ok=True)
binary=R/'bench'
def sha(p):
 h=hashlib.sha256()
 with p.open('rb') as f:
  for b in iter(lambda:f.read(2**20),b''):h.update(b)
 return h.hexdigest()
assert not any(os.environ.get(k) for k in ['GOMAXPROCS','GOGC','GOMEMLIMIT'])
cache=R/'data/synthetic'
cache.mkdir(parents=True,exist_ok=True)
configs=[dict(key=f'scale-{n}',group='scale',n=n,m=10,dataset='') for n in [10000,100000,1000000,2000000,5000000]]
configs += [dict(key=f'granularity-{m}',group='granularity',n=2000000,m=m,dataset='') for m in [1,10,100,1000,5000]]
configs += [dict(key=f'real-{d}',group='real',n=0,m=0,dataset=d) for d in ['coco','clevr','librispeech']]
# Freeze the complete input set before measuring; do not regenerate a missing
# input belonging to an existing measurement.
for cfg in configs:
 if cfg['dataset']:
  assert (R/'data'/(cfg['dataset']+'.json')).exists(), 'Prepare real metadata first'
  continue
 expected=cache/f"dataset_n{cfg['n']}_m{cfg['m']}_s1024.json"
 if not expected.exists():
  assert not (O/'manifest.json').exists(), 'Recorded input missing; restore it before resuming'
  subprocess.run(['go','run','./cmd/generate-dataset','--n',str(cfg['n']),'--m',str(cfg['m']),'--size','1024','--dir',str(cache)],cwd=R.parent,check=True)
paths=list(cache.glob('dataset_*.json'))+list((R/'data').glob('*.json'))
protocol=dict(rounds=10,operations=1000,payload_bytes=1024,load=.75,processes=13,gomaxprocs='default, not set',construction_goroutines='one per category',proof_queries='first 1000 in file order',modify_queries='1000 starting at floor(N/2)',delete_queries='first 1000',insert='1024-byte payload generator; capacity preparation and deletion outside timers',timing='construction through final root; proof object generation only; update through final root',warmup=False,explicit_gc=False,time_budget=None,real_preprocessing='category grouping and lexical within-category order; file content digests computed before measurement',synthetic_input='cached metadata generated from 1024-byte pseudorandom payloads')
manifest=dict(created=time.strftime('%Y-%m-%d %H:%M:%S %z'),platform=platform.platform(),go=subprocess.check_output(['go','version'],text=True).strip(),binary_sha256=sha(binary),configs=configs,protocol=protocol,input_sha256={str(p):sha(p) for p in paths},source_sha256={str(p.relative_to(C)):sha(p) for p in [C/'dense.go',C/'forest.go',C/'core.go',C/'proof_objects.go',C/'forest_per_category.go',C/'cmd/evaluate/main.go',C/'go.mod',C/'go.sum']})
mp=O/'manifest.json'
if mp.exists():
 old=json.loads(mp.read_text());assert old['binary_sha256']==manifest['binary_sha256'] and {Path(p).name:h for p,h in old['input_sha256'].items()}=={Path(p).name:h for p,h in manifest['input_sha256'].items()}
else:mp.write_text(json.dumps(manifest,indent=2,ensure_ascii=False))
for j,cfg in enumerate(configs):
 dest=O/'jobs'/(cfg['key']+'.jsonl')
 if dest.exists():
  rows=[json.loads(s) for s in dest.read_text().splitlines()];assert len(rows)==50 and all(r['checks_passed'] for r in rows);continue
 tmp=dest.with_suffix('.partial')
 cmd=[str(binary),'--out',str(tmp),'--group',cfg['group'],'--repeat','10','--iterations','1000','--size','1024','--data-dir',str(cache)]
 if cfg['dataset']:cmd+=['--real-json',str(R/'data'/(cfg['dataset']+'.json')),'--dataset-name',cfg['dataset']]
 else:cmd+=['--n',str(cfg['n']),'--m',str(cfg['m'])]
 (O/'progress.json').write_text(json.dumps(dict(completed=j,total=len(configs),current=cfg)))
 start=time.time()
 with (O/'logs'/(cfg['key']+'.log')).open('w') as log:p=subprocess.run(cmd,cwd=C,stdout=log,stderr=subprocess.STDOUT)
 if p.returncode:raise RuntimeError(cfg['key'])
 rows=[json.loads(s) for s in tmp.read_text().splitlines()]
 assert len(rows)==50 and all(r['checks_passed'] and r['count']==(1 if r['operation']=='build' else 1000) for r in rows)
 assert all(r['work']['rebuilds']==0 for r in rows if r['operation']=='insert')
 tmp.rename(dest);print(f'{j+1}/{len(configs)} {cfg["key"]} completed in {time.time()-start:.1f}s',flush=True)
(O/'progress.json').write_text(json.dumps(dict(completed=len(configs),total=len(configs),finished=time.time())))
