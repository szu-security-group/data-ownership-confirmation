"""Audit all scheduled conditions, preserving incomplete phases as censored data."""
from pathlib import Path
import collections,hashlib,json,math
ROOT=Path(__file__).resolve().parent;RESULTS=ROOT/'results/paper'
def check_results():
 m=json.loads((RESULTS/'manifest.json').read_text());env=json.loads((ROOT/'results/environment.json').read_text())
 sha=lambda p:hashlib.sha256(p.read_bytes()).hexdigest()
 assert len(m['jobs'])==200
 assert m['steady_mutation_budget_s']==10 and m['timing']['proof_output_encoding_timed']==False
 assert m['timing']['restore_between_operations']==dict(insert=True,modify=False,delete=False)
 assert all(j['group'] in ['scale','real'] and j['layout']=='flat' and j['threads']==4 for j in m['jobs'])
 assert sha(ROOT/'bin/bench')==m['binary_sha256']==env['binary_sha256']
 for p,h in env['source_sha256'].items():assert sha(ROOT/p)==h,p
 for p,h in json.loads((ROOT/'third_party/ADAPTATION_MANIFEST.json').read_text()).items():assert sha(ROOT/'third_party'/p)==h['adapted_sha256'],p
 for p,h in env['dataset_sha256'].items():assert sha(ROOT/'datasets'/p)==h,p
 assert all(f'\t{lib}\t{version}' in m['modules'] for lib,version in [('github.com/cbergoon/merkletree','v0.5.0'),('github.com/cosmos/iavl','v1.3.6'),('github.com/celestiaorg/smt','v0.3.0'),('github.com/ethereum/go-ethereum','v1.17.2')])
 expected={hashlib.sha256(json.dumps(j,sort_keys=True).encode()).hexdigest()[:16]:j for j in m['jobs']}
 actual={p.stem:p for p in (RESULTS/'jobs').glob('*.jsonl')};assert actual.keys()==expected.keys(),f'Incomplete jobs: {len(actual)}/{len(expected)}'
 validations={x['name']:x for x in json.loads((ROOT/'datasets/validation.json').read_text())}
 fingerprints=collections.Counter();groups=collections.Counter();statuses=collections.Counter();kinds=collections.Counter();records=0
 matched_inputs=collections.defaultdict(list)
 matched_requests=collections.defaultdict(list)
 for key,j in expected.items():
  rr=[json.loads(s) for s in actual[key].read_text().splitlines() if s];records+=len(rr);ops=[r['operation'] for r in rr]
  assert j['workload']=='steady' and ops==['build','prove','insert','modify','delete']
  for r in rr:
   assert not {'heap_bytes','alloc_bytes','process_peak_rss_bytes'} & r.keys()
   for f in ['kind','layout','threads','seed','repeat','workload','dataset','group']:assert r[f]==j[f],(key,f)
   assert r['binary_sha256']==m['binary_sha256'] and r['job_id']==key
   assert r['status'] in ['complete','budget_exceeded'];statuses[r['status']]+=1
   if r['status']!='complete':kinds[r['kind']]+=1
   assert r['hash_counts_available']==(j['kind'] in ['ht','smt'])
   assert all(0<=v<2**63 for v in r['work'].values())
   if j['dataset']:
    v=validations[Path(j['dataset']).stem];assert r['n']==v['files'] and r['m']==len(v['categories'])
   else:assert r['n']==j['n'] and r['m']==j['m']
   for f in ['mean_ns','total_ns','p50_ns','p95_ns','p99_ns','max_ns','proof_bytes']:assert math.isfinite(r[f]) and r[f]>=0
   if r['operation']!='build':
    assert r['requested_count']==j['ops'] and 0<r['count']<=j['ops']
    assert (r['count']==j['ops'])==(r['status']=='complete')
    assert r['p50_ns']<=r['p95_ns']<=r['p99_ns']<=r['max_ns']
  pf=RESULTS/'proofs'/(key+'.jsonl');pp=[json.loads(s) for s in pf.read_text().splitlines() if s]
  assert len(pp)==(3 if j['workload']=='steady' and j['repeat']==0 else 0)
  for r in pp:fingerprints[(key,r['url'],r['digest'],r['root'],r['proof'])]+=1
  condition=tuple(j[f] for f in ['group','n','m','dataset','threads','seed','repeat'])
  prefix=True
  for r in rr:
   if r['operation']=='prove':assert r['extra']['proof_output_encoding_timed']==False
   if r['operation'] in ['insert','modify','delete']:
    assert r['extra']['input_prefix_complete']==prefix
    assert r['extra']['restore_between_operations']==(r['operation']=='insert') and r['extra']['postconditions_checked']
    if r['operation']=='insert' and r['kind']=='ht':assert r['work']['rebuilds']==0
    matched_requests[(condition,r['operation'])].append(r['extra']['request_digest'])
    prefix=prefix and r['status']=='complete'
  matched_inputs[condition].append(tuple((r['url'],r['digest']) for r in pp))
  groups[j['group']]+=1
 assert len(matched_requests)==120 and all(len(v)==5 and len(set(v))==1 for v in matched_requests.values()),'Update sequences differ'
 assert len(matched_inputs)==40
 assert all(len(samples)==5 and len(set(samples))==1 for samples in matched_inputs.values()),'Component proof requests differ'
 assert sum(fingerprints.values())==sum(3 for j in m['jobs'] if j['workload']=='steady' and j['repeat']==0)
 report=dict(status='PASS',scheduled_jobs=len(m['jobs']),finished_processes=len(actual),groups=dict(groups),measured_records=records,phase_statuses=dict(statuses),budget_exceeded_by_kind=dict(kinds),sampled_proofs=sum(fingerprints.values()),binary_sha256=m['binary_sha256'],note='Finished processes include explicitly censored phases; no partial phase is reported as a completed workload.')
 return report
if __name__=='__main__':
 report=check_results();(RESULTS/'audit.json').write_text(json.dumps(report,indent=2)+'\n');print(json.dumps(report,indent=2))
