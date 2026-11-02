"""Return only cells whose entire planned set of repetitions completed the phase."""
from pathlib import Path
import json,collections,csv
ROOT=Path(__file__).resolve().parent
KEYS=['group','kind','layout','dataset','n','m','threads','load','workload','operation']
def cell(r):return tuple(r.get(k,'') for k in KEYS)+((r.get('extra') or {}).get('fraction',''),)
def prepare(base,out):
 raw=[];logical=[];censored_jobs=set()
 for p in sorted((base/'jobs').glob('*.jsonl')):
  rr=[json.loads(s) for s in p.read_text().splitlines() if s]
  if rr[0]['layout']!='flat' or rr[0]['group'] not in ['scale','real']:continue
  assert rr[0]['workload']=='steady'
  rr=[dict(r,layout='categorized') for r in rr]
  for r in rr:r['load']=r.get('load',.75) if r['kind']=='ht' else None
  raw.extend(rr);phase_incomplete=any(r['status']!='complete' for r in rr)
  for r in rr:
   z=dict(r,process_workload_complete=not phase_incomplete)
   logical.append(z)
 groups=collections.defaultdict(list)
 for r in logical:groups[cell(r)].append(r)
 coverage=[];formal=[];rejected=[]
 for key,rs in groups.items():
  expected=5
  ok=len(rs)==expected and all(r['status']=='complete' and (r.get('extra') or {}).get('input_prefix_complete',True) for r in rs)
  v=dict(zip(KEYS+['fraction'],key));v.update(expected_runs=expected,recorded_runs=len(rs),complete_runs=sum(r['status']=='complete' for r in rs),complete_input_runs=sum((r.get('extra') or {}).get('input_prefix_complete',True) for r in rs),status='complete' if ok else 'censored',completed_operations=sum(r['count'] for r in rs),requested_operations=sum(r.get('requested_count',0) for r in rs),minimum_operations=min(r['count'] for r in rs),maximum_operations=max(r['count'] for r in rs))
  coverage.append(v)
  if ok:formal.extend(rs)
  else:rejected.append(v)
 out.mkdir(exist_ok=True,parents=True)
 selected_jobs={r['job_id']:r['group'] for r in raw}
 selection=dict(processes=len(selected_jobs),groups=dict(collections.Counter(selected_jobs.values())),records=len(raw),job_ids=sorted(selected_jobs),organization='One authenticated subtree per category; ordered category roots are hashed together.')
 (out/'experiment_scope.json').write_text(json.dumps(selection,indent=2)+'\n')
 with (out/'completion.csv').open('w',newline='') as f:w=csv.DictWriter(f,fieldnames=list(coverage[0]));w.writeheader();w.writerows(coverage)
 (out/'completion.json').write_text(json.dumps(coverage,indent=2)+'\n')
 (out/'censored_cells.json').write_text(json.dumps(rejected,indent=2)+'\n')
 # Partial phase summaries are retained separately and explicitly labelled; never fed to formal plots.
 with (out/'censored_runs.jsonl').open('w') as f:
  for r in logical:
   if r['status']!='complete' or not (r.get('extra') or {}).get('input_prefix_complete',True):
    z=dict(r,exclusion_reason='time_limit' if r['status']!='complete' else 'incomplete_preceding_phase');f.write(json.dumps(z)+'\n')
 return raw,formal,censored_jobs
