from pathlib import Path
import matplotlib
matplotlib.use("Agg")
import matplotlib.pyplot as plt
import numpy as np

# Read the measured means instead of embedding a second copy of the numbers.
import json
rows = json.loads((Path(__file__).resolve().parent / 'output/summary.json').read_text())
def values(operation, divisor):
    return [next(r['mean_ns'] / divisor for r in rows
                 if r['group'] == 'real' and r['dataset'] == name and r['operation'] == operation)
            for name in ['coco', 'clevr', 'librispeech']]
datasets = ['MS COCO', 'CLEVR', 'LibriSpeech']
build_times = values('build', 1e6)
proof_times = values('prove', 1e3)
insert_times = values('insert', 1e3)
delete_times = values('delete', 1e3)
modify_times = values('modify', 1e3)

# === 1. 字体与全局设置 ===
plt.rcParams['font.family'] = 'serif'
plt.rcParams['font.serif'] = ['Times New Roman']
plt.rcParams['mathtext.fontset'] = 'stix'
# 全局基础字体设为 14，保证 ticks 和 legend 默认够大
plt.rcParams['font.size'] = 14

# === 2. 专业配色 ===
colors_build = ['#4C72B0', '#DD8452', '#55A868']
color_proof = '#4C72B0'
color_insert = '#55A868'
color_delete = '#DD8452'
color_modify = '#C44E52'

# === 3. 创建画布 ===
fig, (ax1, ax2) = plt.subplots(1, 2, figsize=(14, 6))
plt.subplots_adjust(wspace=0.25)

# === 4. 绘制左图 ===
ax1.bar(datasets, build_times, color=colors_build, width=0.5,
        edgecolor='black', linewidth=0.8, alpha=0.9)

# [字体设置重点]
# 1. 子图标题 (最大): 18pt
ax1.set_title('(a) ADS Construction', fontsize=18, pad=12)
# 2. 坐标轴标题 (中等): 16pt
ax1.set_ylabel('Time cost (ms)', fontsize=16)
# 3. 刻度标签 (较小): 全局已设为14，或者手动 tick_params(labelsize=14)
ax1.tick_params(axis='both', labelsize=14)

ax1.set_ylim(0, 150)
ax1.grid(axis='y', linestyle='--', alpha=0.5)

# === 5. 绘制右图 ===
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

# [字体设置重点]
ax2.set_title('(b) Proof Path Generation & Updates', fontsize=18, pad=12)
ax2.set_ylabel('Time cost (µs)', fontsize=16)
ax2.set_xticks(x)
ax2.set_xticklabels(datasets, fontsize=14) # 显式设置 X 轴刻度字体
ax2.tick_params(axis='y', labelsize=14)
ax2.set_ylim(0, 80)
ax2.grid(axis='y', linestyle='--', alpha=0.5)

# 4. 图例 (与刻度一致或略小): 14pt
# handlelength=1.5 控制色块长度，handletextpad=0.5 控制色块和文字间距，使图例紧凑
ax2.legend(fontsize=14, loc='upper left', frameon=False,
           handlelength=1.2, handletextpad=0.5)

# === 6. 保存 ===
plt.tight_layout()
output = Path(__file__).resolve().parent / 'output/figures'
output.mkdir(parents=True, exist_ok=True)
plt.savefig(output / 'real_datasets_bars.pdf', dpi=300, bbox_inches='tight')
plt.savefig(output / 'real_datasets_bars.png', dpi=180, bbox_inches='tight')
plt.close(fig)
