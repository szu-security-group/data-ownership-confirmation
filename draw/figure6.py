import matplotlib.pyplot as plt
import numpy as np
from matplotlib.ticker import ScalarFormatter

m_values = [1, 10, 100, 1000, 5000]
build_times = [7735.3, 3475.2, 2466.9, 3019.8, 3437.6]
proof_times = [3.8, 5.8, 13.0, 112.9, 471.2]
add_times = [25.6, 31.6, 37.9, 139.4, 775.2]
del_times = [22.9, 27.9, 32.0, 149.7, 705.1]
mod_times = [23.1, 31.9, 35.0, 132.5, 677.7]

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

line1 = ax1.plot(m_values, build_times, marker='o', markersize=8, 
                 color=color_build, label='ADS Construction (ms)', 
                 linewidth=2.5, zorder=10)
ax1.set_xlabel('Number of Categories ($m$)', fontsize=16)
ax1.set_ylabel('Construction Time (ms)', color=color_build, fontsize=16)
ax1.tick_params(axis='y', labelcolor=color_build, labelsize=14)
ax1.tick_params(axis='x', labelsize=14)
ax1.set_ylim(0, 9000)

ax2 = ax1.twinx()
ax2.set_ylabel('Operation Latency (µs)', color='black', fontsize=16)

line2 = ax2.plot(m_values, proof_times, marker='s', markersize=7, linestyle='--', 
                 color=color_proof, label='Proof Gen (µs)', linewidth=2)
line3 = ax2.plot(m_values, add_times, marker='^', markersize=7, linestyle='--', 
                 color=color_add, label='Addition (µs)', linewidth=2)
line4 = ax2.plot(m_values, del_times, marker='v', markersize=7, linestyle='--', 
                 color=color_del, label='Deletion (µs)', linewidth=2)
line5 = ax2.plot(m_values, mod_times, marker='D', markersize=7, linestyle='--', 
                 color=color_mod, label='Modification (µs)', linewidth=2)

ax2.tick_params(axis='y', labelcolor='black', labelsize=14)
ax2.set_ylim(0, 1000)

ax1.set_xticks(m_values)
ax1.get_xaxis().set_major_formatter(ScalarFormatter())
ax1.tick_params(axis='x', which='minor', bottom=False)
ax1.grid(True, which="major", linestyle='--', alpha=0.5)

lines = line1 + line2 + line3 + line4 + line5
labels_legend = [l.get_label() for l in lines]
ax1.legend(lines, labels_legend, loc='upper center', fontsize=14, 
           frameon=False, ncol=2, handlelength=1.5, handletextpad=0.5)

ax1.text(0.5, 0.78, 'Experimental Setting: $N = 2 \\times 10^6$', 
         transform=ax1.transAxes, 
         fontsize=14, 
         horizontalalignment='center',
         verticalalignment='top')

plt.tight_layout()
plt.savefig('scalability_granularity_top_adjusted.pdf', dpi=300, bbox_inches='tight')
plt.show()
