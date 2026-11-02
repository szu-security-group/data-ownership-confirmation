"""Paper vector figures; match comparison typography, palette and hollow markers."""
from pathlib import Path
import json
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
from matplotlib.ticker import NullLocator
R=Path(__file__).resolve().parent;F=R/'output/figures';F.mkdir(exist_ok=True)
rows=[s for s in json.loads((R/'output/summary.json').read_text())]
plt.rcParams.update({'font.family':'DejaVu Serif','font.size':9,'axes.titlesize':10,'axes.labelsize':9,'legend.fontsize':8,'xtick.labelsize':8,'ytick.labelsize':8,'pdf.fonttype':42,'ps.fonttype':42,'axes.spines.top':False,'axes.spines.right':False,'savefig.dpi':220,'axes.grid':True,'grid.alpha':.2,'lines.linewidth':1.35,'lines.markersize':4})
OPS=['prove','insert','modify','delete'];COLORS=['#0072B2','#009E73','#E69F00','#CC79A7'];MARKERS=['o','^','s','D'];STYLES=['-','-.','--',':'];LABELS=['Proof generation','Insertion','Modification','Deletion']
def value(group,op,n=None,m=None,ds=None):
 matches=[s for s in rows if s['group']==group and s['operation']==op and (n is None or s['n']==n) and (m is None or s['m']==m) and (ds is None or s['dataset']==ds)]
 assert len(matches)==1,(group,op,n,m,ds)
 return matches[0]['mean_ns']
def save(fig,name):
 fig.savefig(F/(name+'.pdf'),bbox_inches='tight',pad_inches=.05)
 fig.savefig(F/(name+'.png'),bbox_inches='tight',pad_inches=.05)
 plt.close(fig)
for g,xs,xlabel,labels in [('scale',[10000,100000,1000000,2000000,5000000],'Number of data items, $N$',['10k','100k','1M','2M','5M']),('granularity',[1,10,100,1000,5000],'Number of categories, $m$',['1','10','100','1000','5000'])]:
 fig,axs=plt.subplots(1,2,figsize=(7.25,2.65),layout='constrained')
 def val(op,x):
  if g=='scale':return value(g,op,n=x)
  return value(g,op,m=x)
 axs[0].plot(xs,[val('build',x)/1e9 for x in xs],marker='o',markerfacecolor='none',color='#0072B2');axs[0].set_ylabel('Time (s)');axs[0].set_title('(a) Construction')
 if g=='scale':axs[0].set_yscale('log')
 for op,color,marker,style,label in zip(OPS,COLORS,MARKERS,STYLES,LABELS):axs[1].plot(xs,[val(op,x)/1e3 for x in xs],marker=marker,markerfacecolor='none',color=color,linestyle=style,label=label)
 axs[1].set_ylabel('Time (µs)');axs[1].set_title('(b) Proof generation and updates');axs[1].set_yscale('log')
 for ax in axs:
  ax.set_xscale('log');ax.set_xlabel(xlabel);ax.set_xticks(xs,labels);ax.xaxis.set_minor_locator(NullLocator())
 fig.legend(*axs[1].get_legend_handles_labels(),loc='lower center',bbox_to_anchor=(.5,1.01),ncol=4,frameon=False)
 save(fig,g)

# Plot the real-dataset measurements.
import subprocess,sys
subprocess.run([sys.executable,str(R/'plot_real.py')],check=True)
