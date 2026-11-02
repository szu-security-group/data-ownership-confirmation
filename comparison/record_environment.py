"""Record a fresh reproduction environment without overwriting frozen provenance."""
from pathlib import Path
import hashlib,json,os,platform,subprocess,time
root=Path(__file__).resolve().parent
output=root/'results/environment.json'
if output.exists():raise SystemExit('Environment already frozen. Use a fresh reproduction copy; do not overwrite provenance of existing measurements.')
files=[p for p in (root/'third_party').rglob('*') if p.is_file()]+list(root.glob('*.go'))+list((root/'cmd').rglob('*.go'))+[root/'go.mod',root/'go.sum',root/'run_experiment.py']
files += list((root/'internal').rglob('*.go'))
files += [root.parent/'go.mod',root.parent/'go.sum']
for folder in ['pkg','utils']:
 files += list((root.parent/folder).rglob('*.go'))
def sha(p):return hashlib.sha256(p.read_bytes()).hexdigest()
def command(*args):return subprocess.check_output(args,text=True).strip()
e=dict(recorded_at=time.strftime('%Y-%m-%d %H:%M:%S %z'),platform=platform.platform(),go=command('go','version'),source_sha256={os.path.relpath(p,root):sha(p) for p in files},dataset_sha256={p.name:sha(p) for p in (root/'datasets').glob('*.jsonl')},binary_sha256=sha(root/'bin/bench'),modules=command('go','version','-m',str(root/'bin/bench')),test_logs={p.name:p.read_text() for p in [root/'results/go-test.log',root/'results/go-race.log'] if p.exists()},config=dict(steady_budget_s=10,groups=['scale','real'],processes=200))
if platform.system()!='Darwin':raise SystemExit('Record CPU environment for this platform before running.')
e.update(cpu_model=command('sysctl','-n','machdep.cpu.brand_string'),memory_bytes=int(command('sysctl','-n','hw.memsize')),logical_cpus=int(command('sysctl','-n','hw.logicalcpu')))
output.parent.mkdir(parents=True,exist_ok=True);output.write_text(json.dumps(e,indent=2)+'\n')
print('Recorded',output)
