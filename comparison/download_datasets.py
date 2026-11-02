"""Download the dataset releases used in the paper."""
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path
import subprocess
import json
import time

ROOT = Path(__file__).resolve().parent / 'datasets'
SOURCES = {
 'train2017.zip': 'https://images.cocodataset.org/zips/train2017.zip',
 'val2017.zip': 'https://images.cocodataset.org/zips/val2017.zip',
 'test2017.zip': 'https://images.cocodataset.org/zips/test2017.zip',
 'annotations_trainval2017.zip': 'https://images.cocodataset.org/annotations/annotations_trainval2017.zip',
 'CLEVR_v1.0.zip': 'https://dl.fbaipublicfiles.com/clevr/CLEVR_v1.0.zip',
 'train-other-500.tar.gz': 'https://openslr.trmal.net/resources/12/train-other-500.tar.gz',
}

def download(pair):
 name, url = pair
 dest = ROOT / name
 if dest.exists():
  return {'file':name,'url':url,'bytes':dest.stat().st_size,'status':'existing'}
 part = ROOT / (name + '.part')
 log = ROOT / (name + '.download.log')
 print('Downloading', name, flush=True)
 with log.open('a') as out:
  r = subprocess.run(['curl','--max-time','21600','--fail','--location','--retry','8',
       '--retry-delay','5','--continue-at','-','--output',str(part),url],stdout=out,stderr=out)
 if r.returncode:
  raise RuntimeError(f'{name}: curl exit {r.returncode}; inspect {log}')
 part.rename(dest)
 print('Completed', name, dest.stat().st_size, flush=True)
 return {'file':name,'url':url,'bytes':dest.stat().st_size,'status':'downloaded'}

if __name__ == '__main__':
 ROOT.mkdir(parents=True,exist_ok=True)
 with ThreadPoolExecutor(max_workers=3) as pool:
  records=list(pool.map(download,SOURCES.items()))
 (ROOT/'sources.json').write_text(json.dumps({'completed_unix':time.time(),'archives':records},indent=2))
