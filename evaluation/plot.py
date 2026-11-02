"""Draw paper Figures 4--6 from evaluation/output/summary.json."""
from pathlib import Path
import argparse
import json
import matplotlib
matplotlib.use('Agg')
import matplotlib.pyplot as plt
import numpy as np
from matplotlib.ticker import NullLocator, FixedLocator

ROOT = Path(__file__).resolve().parent
HT_DATA = []
OPS = ['prove', 'insert', 'modify', 'delete']
OP_COLORS = ['#0072B2', '#009E73', '#E69F00', '#CC79A7']
OP_MARKERS = ['o', '^', 's', 'D']
OP_STYLES = ['-', '-.', '--', ':']
OP_LABELS = ['Proof generation', 'Insertion', 'Modification', 'Deletion']
datasets = ['coco', 'clevr', 'librispeech']
names = ['MS COCO', 'CLEVR', 'LibriSpeech']

def ht_value(group, operation, **filters):
    matches = [r for r in HT_DATA if r['group'] == group
               and r['operation'] == operation
               and all(r.get(k) == v for k, v in filters.items())]
    assert len(matches) == 1, (group, operation, filters)
    return matches[0]['mean_ns']

def compact_style():
    plt.rcdefaults()
    plt.rcParams.update({
        'font.family': 'DejaVu Serif', 'font.size': 7,
        'axes.labelsize': 7, 'xtick.labelsize': 6.5, 'ytick.labelsize': 6.5,
        'legend.fontsize': 6.6, 'pdf.fonttype': 42,
        'axes.spines.top': False, 'axes.spines.right': False,
        'axes.linewidth': .6, 'lines.linewidth': .9, 'lines.markersize': 2.6,
        'xtick.major.width': .5, 'ytick.major.width': .5,
        'xtick.major.size': 2, 'ytick.major.size': 2,
        'axes.labelpad': 2, 'xtick.major.pad': 2, 'ytick.major.pad': 2,
        'mathtext.fontset': 'dejavuserif', 'figure.dpi': 180,
        'savefig.dpi': 240,
    })

def real_values(op, divisor):
    return [ht_value('real', op, dataset=d) / divisor for d in datasets]

def draw_real_panel(ax, index):
    if index == 0:
        ax.bar(range(3), real_values('build', 1e6), width=.5,
               color='#4C72B0', edgecolor='black', linewidth=.4, alpha=.9)
        ax.set(ylabel='Time cost (ms)', ylim=(0,150), yticks=[0,50,100,150])
    else:
        # Grouped points: horizontal offsets distinguish the operations;
        # data points from different datasets are not connected.
        for offset, op, color, marker, label in zip(
                [-1.5,-.5,.5,1.5], ['prove','insert','delete','modify'],
                ['#4C72B0','#55A868','#DD8452','#C44E52'],
                ['o','^','D','s'], ['Proof Gen','Insert','Delete','Modify']):
            ax.plot(np.arange(3)+offset*.18, real_values(op, 1e3),
                    linestyle='none', marker=marker, markersize=3.2,
                    markerfacecolor='white', markeredgewidth=.7,
                    color=color, label=label)
        ax.set_yscale('log')
        ax.set(ylabel='Time cost (µs)', ylim=(.2,150))
        ax.set_yticks([1,10,100], ['1','10','100'])
        ax.yaxis.set_minor_locator(NullLocator())
        ax.set_xlim(-.55,2.55)
        ax.legend(loc='upper left', fontsize=6, frameon=False,
                  handlelength=1, handletextpad=.4, borderpad=.1, labelspacing=.18)
    ax.spines['top'].set_visible(True)
    ax.spines['right'].set_visible(True)
    ax.set_xticks(range(3), names)
    ax.tick_params(axis='x', labelsize=5.5)
    ax.set_axisbelow(True)
    ax.grid(axis='y', linestyle='--', alpha=.35, linewidth=.4)

def plot_ht_scalability(group):
    compact_style()
    is_scale = group == 'scale'
    xs = [10000,100000,1000000,2000000,5000000] if is_scale else [1,10,100,1000,5000]
    key = 'n' if is_scale else 'm'
    xlabel = 'Data items, $N$' if is_scale else 'Categories, $m$'
    def value(op, x):
        return ht_value(group, op, **{key:x})
    fig = plt.figure(figsize=(3.53, 1.98))
    axes = [fig.add_axes([.105,.29,.36,.54]), fig.add_axes([.625,.29,.36,.54])]
    axes[0].plot(xs,[value('build',x)/1e9 for x in xs],marker='o',
                 markerfacecolor='white',markeredgewidth=.65,color=OP_COLORS[0])
    axes[0].set_ylabel('Time (s)')
    if is_scale:
        axes[0].set_yscale('log')
        axes[0].set_ylim(.0019,1.8)
        axes[0].yaxis.set_major_locator(FixedLocator([.01,.1,1]))
    else:
        axes[0].set_ylim(.15,2.15)
        axes[0].set_yticks([.5,1,1.5,2])
    for op,c,m,ls,label in zip(OPS,OP_COLORS,OP_MARKERS,OP_STYLES,OP_LABELS):
        axes[1].plot(xs,[value(op,x)/1e3 for x in xs],marker=m,
                     markerfacecolor='white',markeredgewidth=.65,
                     color=c,linestyle=ls,label=label)
    axes[1].set_ylabel('Time (µs)')
    axes[1].set_yscale('log')
    axes[1].set_ylim((.2,12) if is_scale else (.4,420))
    axes[1].yaxis.set_major_locator(FixedLocator([1,10] if is_scale else [1,10,100]))
    for ax in axes:
        ax.set_xscale('log')
        ax.set_xlabel(xlabel)
        ticks = [10000,100000,1000000,5000000] if is_scale else xs
        labels = ['10k','100k','1M','5M'] if is_scale else ['1','10','100','1k','5k']
        ax.set_xticks(ticks,labels)
        ax.xaxis.set_minor_locator(NullLocator())
        ax.yaxis.set_minor_locator(NullLocator())
        ax.set_xlim(xs[0]/1.35,xs[-1]*1.35)
        ax.grid(True,alpha=.23,linewidth=.45)
    fig.legend(*axes[1].get_legend_handles_labels(),loc='upper center',
               bbox_to_anchor=(.5,1.025),ncol=2,frameon=False,columnspacing=1.8,
               handlelength=2,handletextpad=.5,labelspacing=.25)
    fig.text(.285,.05,'(a) Construction',ha='center',fontsize=7)
    fig.text(.805,.05,'(b) Runtime Operations',ha='center',fontsize=7)
    return fig


def save(fig, output, name, tight=False):
    options = dict(bbox_inches='tight', pad_inches=.05) if tight else {}
    fig.savefig(output / (name + '.pdf'), **options)
    fig.savefig(output / (name + '.png'), dpi=220, **options)


def plot_real(output):
    compact_style()
    plt.rcParams.update({'font.family': 'serif',
                         'font.serif': ['Times New Roman', 'DejaVu Serif'],
                         'mathtext.fontset': 'stix'})
    for index, letter in enumerate('ab'):
        fig = plt.figure(figsize=(1.59, 1.45))
        draw_real_panel(fig.add_axes([.245, .18, .735, .795]), index)
        save(fig, output, f'performance_log_{letter}')
        plt.close(fig)
    fig = plt.figure(figsize=(3.53, 1.72))
    axes = [fig.add_axes([.10, .29, .36, .67]),
            fig.add_axes([.62, .29, .36, .67])]
    for index, ax in enumerate(axes):
        draw_real_panel(ax, index)
    fig.text(.28, .055, '(a) ADS Construction', ha='center', fontsize=7)
    fig.text(.79, .055, '(b) Proof Path Generation & Updates', ha='center', fontsize=6)
    save(fig, output, 'real_datasets', tight=True)
    plt.close(fig)


def plot_scale(group, output):
    # The notebook preview and manuscript panels use the same axes and data.
    fig = plot_ht_scalability(group)
    save(fig, output, group, tight=True)
    handles, labels = fig.axes[1].get_legend_handles_labels()
    for index, letter in enumerate('ab'):
        # Export each axes at its exact manuscript canvas size.
        fig.set_size_inches(1.73, 1.45)
        for i, ax in enumerate(fig.axes):
            ax.set_visible(i == index)
        for text in fig.texts:
            text.set_visible(False)
        for legend in fig.legends:
            legend.set_visible(False)
        fig.axes[index].set_position([.24, .24, .73, .73])
        save(fig, output, f'{group}_{letter}')
    plt.close(fig)
    fig = plt.figure(figsize=(3.53, .29))
    fig.legend(handles, labels, loc='center', ncol=2, frameon=False,
               columnspacing=1.8, handlelength=2, handletextpad=.5,
               borderaxespad=0, labelspacing=.25)
    save(fig, output, 'operation_legend')
    plt.close(fig)


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--group', choices=['all', 'real', 'scale', 'granularity'], default='all')
    parser.add_argument('--summary', type=Path, default=ROOT / 'output/summary.json')
    parser.add_argument('--output', type=Path, default=ROOT / 'output/figures')
    args = parser.parse_args(argv)
    if not args.summary.is_file():
        parser.error('Summary not found. Run evaluation/run.py and evaluation/analyze.py first, '
                     'or pass --summary PATH. For the paper-data snapshot, open draw/paper_experiment_figures.ipynb.')
    global HT_DATA
    HT_DATA = json.loads(args.summary.read_text())
    args.output.mkdir(parents=True, exist_ok=True)
    groups = ['real', 'scale', 'granularity'] if args.group == 'all' else [args.group]
    for group in groups:
        if group == 'real':
            plot_real(args.output)
        else:
            plot_scale(group, args.output)
    print(f'Figures written to {args.output}')


if __name__ == '__main__':
    main()
