"""Sequential, resumable benchmark: one fresh process per condition/run.
No parallel performance jobs: all variants see the same CPU budget.
"""
from pathlib import Path
import argparse, subprocess, json, random, time, hashlib, platform, os, re
ROOT=Path(__file__).resolve().parent
p=argparse.ArgumentParser();p.add_argument('--profile',choices=['smoke','paper'],default='paper');p.add_argument('--binary',type=Path,default=ROOT/'bin/bench');a=p.parse_args()
out=ROOT/'results'/a.profile;out.mkdir(parents=True,exist_ok=True)
(out/'jobs').mkdir(exist_ok=True);(out/'proofs').mkdir(exist_ok=True);(out/'traces').mkdir(exist_ok=True)
kinds=['ht','merkle','iavl','smt','mpt'];jobs=[]
def add(group,kind,layout,n,m=10,threads=None,workload='steady',dataset='',ops=10000,seeds=range(1,6),repeats=range(1)):
 for seed in seeds:
  for repeat in repeats:
   jobs.append(dict(group=group,kind=kind,layout=layout,n=n,m=m,threads=threads or (1 if layout=='global' else 4),workload=workload,dataset=dataset,ops=ops,seed=seed,repeat=repeat))
if a.profile=='smoke':
 for k in kinds:
  for l in ['flat']: add('smoke',k,l,1000,m=5,ops=50,seeds=[1],repeats=[0])
else:
 for k in kinds:
  for l in ['flat']:
   for n in [10000,100000,1000000,2000000,5000000]:add('scale',k,l,n)
   for ds in ['coco','clevr','librispeech']:add('real',k,l,0,dataset=str(ROOT/'datasets'/f'{ds}.jsonl'))
# A fixed shuffled execution order distributes conditions over time.
random.Random(20260924).shuffle(jobs)
binary=a.binary.resolve()
manifest={'profile':a.profile,'created_unix':time.time(),'platform':platform.platform(),'python':platform.python_version(),'go':subprocess.check_output(['go','version'],text=True).strip(),'cpu_count':os.cpu_count(),'binary_sha256':hashlib.sha256(binary.read_bytes()).hexdigest(),'jobs':jobs,'steady_mutation_budget_s':10,'timing':{'initialization_included':True,'category_routing':'URL','proof_output_encoding_timed':False,'restore_between_operations':{'insert':True,'modify':False,'delete':False},'insertion_preparation':'one insert/delete pair per requested category outside timing'},'modules':subprocess.check_output(['go','version','-m',str(binary)],text=True)}
manifest_path=out/'manifest.json'
if manifest_path.exists():
 old=json.loads(manifest_path.read_text())
 if old['binary_sha256']!=manifest['binary_sha256'] or old['jobs']!=manifest['jobs']:raise SystemExit('Binary changed; choose a new results directory to avoid mixing implementations.')
else:manifest_path.write_text(json.dumps(manifest,indent=2))
start=time.time();completed=0
for j in jobs:
 key=hashlib.sha256(json.dumps(j,sort_keys=True).encode()).hexdigest()[:16];dest=out/'jobs'/f'{key}.jsonl'
 if dest.exists():completed+=1;continue
 tmp=dest.with_suffix('.partial');tmp.write_text('');proof=out/'proofs'/f'{key}.jsonl';proof.write_text('');trace=out/'traces'/f'{key}.jsonl'
 cmd=[str(binary),'--out',str(tmp)]
 cmd += ['--steady-budget','10']
 for k in ['kind','layout','n','m','threads','workload','ops','seed','repeat']:cmd+=['--'+k,str(j[k])]
 if 'load' in j:cmd+=['--load',str(j['load'])]
 if j['dataset']:cmd+=['--dataset',j['dataset']]
 # 3 real proof samples per condition per independent seed; no synthetic paths.
 if j['workload']=='steady' and j['repeat']==0:cmd+=['--proof-out',str(proof)]
 status={'completed':completed,'total':len(jobs),'current':j,'started_unix':time.time(),'elapsed_s':time.time()-start}
 (out/'progress.json').write_text(json.dumps(status,indent=2))
 before=time.time();r=subprocess.run(cmd,cwd=ROOT,capture_output=True,text=True)
 if r.returncode:
  (out/f'error-{key}.txt').write_text(r.stdout+'\n'+r.stderr);raise SystemExit(f'Failed {j}: {r.stderr}')
 # Attach controlled condition metadata; no process resource measurement.
 rows=[dict(json.loads(s),group=j['group'],job_id=key,binary_sha256=manifest['binary_sha256'],load=j.get('load',.75)) for s in tmp.read_text().splitlines()]
 tmp.write_text('\n'.join(json.dumps(x) for x in rows)+'\n');tmp.rename(dest);completed+=1
 print(f'{completed}/{len(jobs)} {j["group"]} {j["kind"]}/{j["layout"]} N={j["n"]} m={j["m"]} seed={j["seed"]} rep={j["repeat"]} {time.time()-before:.1f}s',flush=True)
(out/'progress.json').write_text(json.dumps({'completed':completed,'total':len(jobs),'finished_unix':time.time(),'elapsed_s':time.time()-start},indent=2))
