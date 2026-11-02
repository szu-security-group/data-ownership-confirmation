"""Generate the scale and real-data figures from complete measured cells."""
from pathlib import Path
import json,csv,math,collections,os,hashlib
import numpy as np
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
from matplotlib.lines import Line2D
from matplotlib.ticker import LogLocator,NullFormatter
ROOT=Path(__file__).resolve().parent; IN=Path(os.environ.get('ADS_RESULTS',ROOT/'results/paper')); OUT=Path(os.environ.get('ADS_OUTPUT',ROOT/'output')); OUT.mkdir(exist_ok=True,parents=True)
if 'ADS_RESULTS' not in os.environ:
 from audit_results import check_results
 check_results()
FIG=OUT/'figures';FIG.mkdir(exist_ok=True)
KINDS=['ht','merkle','iavl','smt','mpt'];LABEL={'ht':'HT-Merkle (ours)','merkle':'Merkle','iavl':'IAVL','smt':'SMT','mpt':'MPT'}
COLORS=['#0072B2','#E69F00','#009E73','#CC79A7','#D55E00'];MARKERS=['o','s','^','D','v']
STYLES=['-','--','-.',':',(0,(3,1,1,1))]
plt.rcParams.update({'font.family':'DejaVu Serif','font.size':9,'axes.titlesize':10,'axes.labelsize':9,'legend.fontsize':8,'xtick.labelsize':8,'ytick.labelsize':8,'pdf.fonttype':42,'ps.fonttype':42,'axes.spines.top':False,'axes.spines.right':False,'savefig.dpi':220,'axes.grid':True,'grid.alpha':.2,'lines.linewidth':1.35,'lines.markersize':4})
rows=[]
for p in sorted((IN/'jobs').glob('*.jsonl')):rows.extend(json.loads(x) for x in p.read_text().splitlines() if x)
from prepare_results import prepare
raw_rows,rows,censored_jobs=prepare(IN,OUT)
def sel(data=rows,**kw):return [r for r in data if all(r.get(k)==v for k,v in kw.items())]
def stat(data,field):
 seeds=collections.defaultdict(list)
 for r in data:
  if isinstance(r.get(field),(float,int)):seeds[r['seed']].append(r[field])
 x=np.array([np.mean(v) for v in seeds.values()]);mean=float(np.mean(x)) if len(x) else math.nan
 # Student intervals over independent seed means, never over 10k operations.
 t={2:12.706,3:4.303,4:3.182,5:2.776}.get(len(x),1.96)
 err=float(t*np.std(x,ddof=1)/math.sqrt(len(x))) if len(x)>1 else 0.
 return mean,err,len(x)
def save(fig,name):
 selected=os.environ.get('ADS_FIGURES','')
 if selected and name not in selected.split(','):
  plt.close(fig);return
 fig.savefig(FIG/(name+'.pdf'),bbox_inches='tight',pad_inches=.05)
 fig.savefig(FIG/(name+'.png'),bbox_inches='tight',pad_inches=.05)
 plt.close(fig)
def legend(fig,axes):
 entries={}
 for ax in axes.flat:
  hs,ls=ax.get_legend_handles_labels()
  for h,l in zip(hs,ls):entries.setdefault(l,h)
 labels=[LABEL[k] for k in KINDS if LABEL[k] in entries]
 fig.legend([entries[l] for l in labels],labels,loc='lower center',ncol=3,bbox_to_anchor=(.5,1.01),frameon=False)
def line(ax,data,xfield,field,div=1):
 for k,color,marker,style in zip(KINDS,COLORS,MARKERS,STYLES):
  rs=sel(data,kind=k);xs=sorted({r[xfield] for r in rs});ys=[];es=[]
  for x in xs:
   y,e,_=stat([r for r in rs if r[xfield]==x],field);ys.append(y/div);es.append(e/div)
  if not xs:continue
  ax.plot(xs,ys,color=color,marker=marker,markerfacecolor='none',linestyle=style,label=LABEL[k])
 ax.set_xscale('log');ax.set_xlabel('Number of data items, $N$')
for layout in ['categorized']:
 fig=plt.figure(figsize=(7.25,4.7),layout='constrained')
 grid=fig.add_gridspec(2,6)
 axs=np.array([fig.add_subplot(grid[0,:3]),fig.add_subplot(grid[0,3:]),fig.add_subplot(grid[1,:2]),fig.add_subplot(grid[1,2:4]),fig.add_subplot(grid[1,4:])])
 for ax,op,field,div,title,label in zip(axs.flat,['build','prove','insert','modify','delete'],['mean_ns']*5,[1e9,1e3,1e3,1e3,1e3],['(a) Construction','(b) Proof generation','(c) Insertion','(d) Modification','(e) Deletion'],['Time (s)','Time (µs)','Time (µs)','Time (µs)','Time (µs)']):
  line(ax,sel(group='scale',layout=layout,operation=op), 'n',field,div);ax.set_ylabel(label);ax.set_title(title);ax.set_yscale('log')
 legend(fig,axs);save(fig,'scale')
fig,axs=plt.subplots(2,3,figsize=(7.25,4.5),layout='constrained')
for ci,ds in enumerate(['coco','clevr','librispeech']):
 for ri,op in enumerate(['build','prove']):
  ax=axs[ri,ci];rs=[r for r in sel(group='real',layout='categorized',operation=op) if Path(r['dataset']).stem==ds];div=1e6 if op=='build' else 1e3
  for j,(k,col) in enumerate(zip(KINDS,COLORS)):
   y,e,_=stat(sel(rs,kind=k),'mean_ns');ax.plot(j,y/div,color=col,marker=MARKERS[j],markerfacecolor='none',markersize=5,linestyle='none')
  ax.set_xticks(range(5),['HT','Merkle','IAVL','SMT','MPT']);ax.set_ylabel('Build time (ms)' if op=='build' else 'Proof time (µs)');ax.set_title(ds.upper() if ri==0 else '');ax.set_yscale('log')
save(fig,'real_datasets')
# Machine-readable aggregates, preserving workload and topology dimensions.
groups=collections.defaultdict(list)
keys=['group','kind','layout','dataset','n','m','threads','load','workload','operation']
for r in rows:
 if r['operation']=='verify':continue # Local validation is not a comparative performance metric.
 extra=r.get('extra') or {};key=tuple(r.get(k,'') for k in keys)+(extra.get('fraction',''),);groups[key].append(r)
aggregates=[]
for key,rr in sorted(groups.items(),key=lambda x:str(x[0])):
 v=dict(zip(keys+['fraction'],key));v['runs']=len(rr);v['seeds']=len({r['seed'] for r in rr});v['process_workload_complete']=all(r['process_workload_complete'] for r in rr)
 for f in ['mean_ns','p50_ns','p95_ns','p99_ns','max_ns','total_ns']:
  applicable=not ((f.startswith('p') and f.endswith('_ns') or f=='max_ns') and v['operation']=='build')
  if applicable:
   y,e,s=stat(rr,f);v[f]=y;v[f+'_ci95']=e
  else:v[f]=None;v[f+'_ci95']=None
 v['hash_counts_available']=all(r['hash_counts_available'] for r in rr) and v['operation'] not in ['verify']
 for f in ['hash_calls','hash_bytes','rebuilds','rehashed_items','rotations','probes']:
  samples=[dict(seed=r['seed'],value=r['work'].get(f,0)) for r in rr]
  if v['operation'] in ['verify'] or v['kind'] in ['merkle','iavl','mpt']:
   v[f]=None;v[f+'_ci95']=None
  else:
   y,e,s=stat(samples,'value');v[f]=y;v[f+'_ci95']=e
 aggregates.append(v)
with (OUT/'summary.csv').open('w',newline='') as f:
 w=csv.DictWriter(f,fieldnames=list(aggregates[0]) if aggregates else keys);w.writeheader();w.writerows(aggregates)
(OUT/'summary.json').write_text(json.dumps(aggregates,indent=2))
# Paired ratios use independent seed means. Values >1 favor HT for cost metrics.
effects=[]
for n in [1000000,5000000]:
 for layout in ['categorized']:
  for op,field in [('build','mean_ns'),('prove','mean_ns'),('insert','mean_ns'),('modify','mean_ns'),('delete','mean_ns')]:
   def seed_means(kind):
    d=collections.defaultdict(list)
    for r in sel(group='scale',kind=kind,layout=layout,n=n,operation=op):d[r['seed']].append(r[field])
    return {s:float(np.mean(v)) for s,v in d.items()}
   reference=seed_means('ht')
   for kind in KINDS[1:]:
    alternative=seed_means(kind);seeds=sorted(reference.keys()&alternative.keys())
    logs=np.array([math.log(alternative[s]/reference[s]) for s in seeds if reference[s]>0 and alternative[s]>0])
    if len(logs)!=5:continue
    mean=float(logs.mean());error=2.776*float(logs.std(ddof=1))/math.sqrt(5)
    effects.append(dict(n=n,layout=layout,operation=op,metric=field,baseline=kind,reference='ht',seeds=5,ratio=math.exp(mean),ci95_low=math.exp(mean-error),ci95_high=math.exp(mean+error)))
with (OUT/'effect_sizes.csv').open('w',newline='') as f:
 w=csv.DictWriter(f,fieldnames=list(effects[0]) if effects else ['n']);w.writeheader();w.writerows(effects)
print('Analyzed',len(rows),'records;',len(list(FIG.glob('*.pdf'))),'PDF figures')
