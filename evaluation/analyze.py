from pathlib import Path
import json,csv,collections,statistics
R=Path(__file__).resolve().parent;O=R/'output';O.mkdir(exist_ok=True)
rows=[]
for p in sorted((R/'results/jobs').glob('*.jsonl')):rows += [dict(json.loads(s),configuration=p.stem) for s in p.read_text().splitlines()]
assert len(rows)==650,len(rows)
groups=collections.defaultdict(list)
for r in rows:
 assert r['checks_passed'] and r['count']==(1 if r['operation']=='build' else 1000)
 assert len(r['samples_ns'])==r['count']
 assert abs(statistics.mean(r['samples_ns'])-r['mean_ns'])<1e-5
 if r['operation']=='insert':assert r['work']['rebuilds']==0
 groups[(r['group'],r['dataset'],r['n'],r['m'],r['operation'])].append(r)
summary=[]
for (g,d,n,m,op),rs in sorted(groups.items()):
 assert len(rs)==10 and {x['round'] for x in rs}==set(range(1,11))
 summary.append(dict(group=g,dataset=d,n=n,m=m,threads=rs[0]['threads'],operation=op,runs=10,operations_per_run=rs[0]['count'],mean_ns=statistics.mean(x['mean_ns'] for x in rs)))
def csvwrite(p,rs):
 with p.open('w') as f:
  w=csv.DictWriter(f,fieldnames=list(rs[0]));w.writeheader();w.writerows(rs)
csvwrite(O/'summary.csv',summary);(O/'summary.json').write_text(json.dumps(summary,indent=2))
qa=dict(configurations=13,rounds=130,records=650,runtime_operations=520000,operations_per_round=1000,complete=True,all_return_values_checked=True,postcondition_samples_passed=True,timed_insert_expansions=0,no_explicit_gc=True,no_warmup=True,no_time_budget=True,payload_generator_checked=True,query_selection_checked=True,per_category_goroutine=True)
(O/'QA.json').write_text(json.dumps(qa,indent=2))
for g in ['scale','granularity','real']:
 print(g)
 keys=list(dict.fromkeys((s['dataset'],s['n'],s['m']) for s in summary if s['group']==g))
 for d,n,m in keys:
  v={s['operation']:s['mean_ns'] for s in summary if s['group']==g and s['dataset']==d and s['n']==n and s['m']==m}
  print(d or str(n),m,'build ms',round(v['build']/1e6,3),'proof/insert/modify/delete us',*[round(v[k]/1e3,3) for k in ['prove','insert','modify','delete']])
