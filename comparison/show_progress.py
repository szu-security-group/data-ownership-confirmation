"""Read-only progress summary; real-data cardinality is resolved from validation."""
from pathlib import Path
import json
root=Path(__file__).resolve().parent
p=json.loads((root/'results/paper/progress.json').read_text())
j=p.get('current',{})
if j:
 dataset=j.get('dataset','')
 if dataset:
  name=Path(dataset).stem
  d=next(x for x in json.loads((root/'datasets/validation.json').read_text()) if x['name']==name)
  detail=f"{name}, N={d['files']:,}, m={len(d['categories'])}"
 else:detail=f"N={j['n']:,}, m={j['m']}"
 print(f"实验进度 {p['completed']:,}/{p['total']:,}；{j['group']} {j['kind']}/{j['layout']}，{detail}，seed={j['seed']}，repeat={j['repeat']}。")
else:print(f"性能测量完成：{p['completed']:,}/{p['total']:,}。")
s=root/'results/postprocess.json'
if s.exists():
 q=json.loads(s.read_text())
 if q['stage']!='waiting_for_performance':print('后处理：',q)
