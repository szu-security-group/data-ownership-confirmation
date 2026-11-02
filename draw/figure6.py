"""Draw Figure 6 using the evaluation plotting implementation."""
from pathlib import Path
import runpy
import sys

if __name__ == '__main__':
    script = Path(__file__).resolve().parents[1] / 'evaluation/plot.py'
    sys.argv = [str(script), '--group', 'granularity', *sys.argv[1:]]
    runpy.run_path(str(script), run_name='__main__')
