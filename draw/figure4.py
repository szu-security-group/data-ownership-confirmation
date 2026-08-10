import matplotlib.pyplot as plt
import numpy as np

datasets = ['MS COCO', 'CLEVR', 'LibriSpeech']
build_times = [396.38, 207.99, 161.56]
proof_times = [1.67, 1.59, 94.72]
insert_times = [11.28, 9.09, 118.30]
delete_times = [11.57, 10.17, 117.37]
modify_times = [14.26, 16.24, 113.24]

plt.rcParams['font.family'] = 'serif'
plt.rcParams['font.serif'] = ['Times New Roman']
plt.rcParams['mathtext.fontset'] = 'stix'
plt.rcParams['font.size'] = 14 

colors_build = ['#4C72B0', '#DD8452', '#55A868']
color_proof = '#4C72B0'
color_insert = '#55A868'
color_delete = '#DD8452'
color_modify = '#C44E52'

fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(14, 6))
plt.subplots_adjust(wspace=0.25)

ax1.bar(datasets, build_times, color=colors_build, width=0.5, 
        edgecolor='black', linewidth=0.8, alpha=0.9)

ax1.set_title('(a) ADS Construction', fontsize=18, pad=12)
ax1.set_ylabel('Time cost (ms)', fontsize=16)
ax1.tick_params(axis='both', labelsize=14) 

ax1.set_ylim(0, 500)
ax1.grid(axis='y', linestyle='--', alpha=0.5)

x = np.arange(len(datasets))
width = 0.2

ax2.bar(x - 1.5*width, proof_times, width, label='Proof Gen', 
        color=color_proof, edgecolor='black', linewidth=0.8, alpha=0.9)
ax2.bar(x - 0.5*width, insert_times, width, label='Insert', 
        color=color_insert, edgecolor='black', linewidth=0.8, alpha=0.9)
ax2.bar(x + 0.5*width, delete_times, width, label='Delete', 
        color=color_delete, edgecolor='black', linewidth=0.8, alpha=0.9)
ax2.bar(x + 1.5*width, modify_times, width, label='Modify', 
        color=color_modify, edgecolor='black', linewidth=0.8, alpha=0.9)

ax2.set_title('(b) Proof Path Generation & Updates', fontsize=18, pad=12)
ax2.set_ylabel('Time cost (µs)', fontsize=16)
ax2.set_xticks(x)
ax2.set_xticklabels(datasets, fontsize=14)
ax2.tick_params(axis='y', labelsize=14)
ax2.set_ylim(0, 150)
ax2.grid(axis='y', linestyle='--', alpha=0.5)

ax2.legend(fontsize=14, loc='upper left', frameon=False, 
           handlelength=1.2, handletextpad=0.5)

plt.tight_layout()
plt.savefig('performance_final.pdf', dpi=300, bbox_inches='tight')
plt.show()
