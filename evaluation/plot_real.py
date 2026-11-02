"""Draw Figure 4 from the measured HT summary."""
import sys
from plot import main

if __name__ == '__main__':
    main(['--group', 'real', *sys.argv[1:]])
