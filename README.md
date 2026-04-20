# Design an Experimental PoS DAG-based Blockchain Consensual Protocol
Author: Tomáš Hladký

### Required tools
- Docker
- Docker Compose plugin
- Python 3.10+

### Usage
To run protcol simuation, follow these steps:

0. Have docker engine running

1. Create virtual environemnt for python (optional but recommended) switch into virtual environemnt
```bash
python3 -m venv /path/to/new/virtual/environment
```

2. Activate virtual environment
```bash
source myenv/bin/activate
```

3. Install python modules
```bash
pip install -r requirements.txt
```

4. Start protocol simulation using initalization script
```bash
python3 init.py
```

Alternatively, prompts can be skipped by loading lastly used values from `setup.yaml` using argument `-n` (`python3 init.py -n`).
