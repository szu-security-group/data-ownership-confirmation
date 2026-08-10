import matplotlib.pyplot as plt
import numpy as np

n_values = [10000, 100000, 1000000, 2000000, 5000000]
build_times = [17.79, 84.22, 1719.25, 3475.18, 8800.07]
proof_times = [1.55, 3.04, 5.14, 5.79, 7.95]
add_times = [11.31, 17.41, 24.92, 31.58, 42.70]
del_times = [8.99, 14.57, 20.96, 27.91, 42.43]
mod_times = [10.53, 10.63, 23.91, 31.85, 52.00]

plt.rcParams['font.family'] = 'serif'
plt.rcParams['font.serif'] = ['Times New Roman']
plt.rcParams['mathtext.fontset'] = 'stix'
plt.rcParams['font.size'] = 14

color_build = '#4C72B0'
color_proof = '#8172B3'
color_add   = '#55A868'
color_del   = '#DD8452'
color_mod   = '#C44E52'

fig, ax1 = plt.subplots(figsize=(10, 6))
ax1.set_xscale('log')

line1 = ax1.plot(n_values, build_times, marker='o', markersize=8, 
                 color=color_build, label='ADS Construction (ms)', 
                 linewidth=2.5, zorder=10)
ax1.set_xlabel('Number of Items ($N$)', fontsize=16)
ax1.set_ylabel('Construction Time (ms)', color=color_build, fontsize=16)
ax1.tick_params(axis='y', labelcolor=color_build, labelsize=14)
ax1.tick_params(axis='x', labelsize=14)
ax1.set_ylim(0, 10000)

ax2 = ax1.twinx()
ax2.set_ylabel('Operation Latency (µs)', color='black', fontsize=16)

line2 = ax2.plot(n_values, proof_times, marker='s', markersize=7, linestyle='--', 
                 color=color_proof, label='Proof Gen (µs)', linewidth=2)
line3 = ax2.plot(n_values, add_times, marker='^', markersize=7, linestyle='--', 
                 color=color_add, label='Addition (µs)', linewidth=2)
line4 = ax2.plot(n_values, del_times, marker='v', markersize=7, linestyle='--', 
                 color=color_del, label='Deletion (µs)', linewidth=2)
line5 = ax2.plot(n_values, mod_times, marker='D', markersize=7, linestyle='--', 
                 color=color_mod, label='Modification (µs)', linewidth=2)
ax2.tick_params(axis='y', labelcolor='black', labelsize=14)
ax2.set_ylim(0, 60)

ax1.set_xticks(n_values)
labels = ['10k', '100k', '1M', '2M', '5M']
ax1.set_xticklabels(labels, fontsize=14)
ax1.tick_params(axis='x', which='minor', bottom=False)
ax1.grid(True, which="major", linestyle='--', alpha=0.5)

lines = line1 + line2 + line3 + line4 + line5
labels_legend = [l.get_label() for l in lines]
ax1.legend(lines, labels_legend, loc='upper left', fontsize=14, 
           frameon=False, handlelength=1.5, handletextpad=0.5)

ax1.text(0.02, 0.65, 'Experimental Setting: $m=10$', 
         transform=ax1.transAxes, 
         fontsize=14, 
         horizontalalignment='left',
         verticalalignment='top')

plt.tight_layout()
plt.savefig('scalability_final_clean_text.pdf', dpi=300, bbox_inches='tight')
plt.show()
