# GPU Server Acceptance Console (local web + SSH remote execution)

[中文](README.md) | [English](README.en.md)

An acceptance testing tool for GPU server delivery. **The console runs only on your local
machine**; every acceptance script is pushed to the target server over SSH and executed as
`bash -s` / `<python> -`. **Nothing is deployed on the server** — no UI, no service, no resident
program (the burn-in workload runs inside a tmux session on the server, which is inherently
required by the test itself). No server address is built in: fill it in on the UI at first run,
or provide it via environment variables.

```
Local machine (Windows / Linux / macOS)   Target GPU server
┌──────────────────────┐   ssh -i key   ┌──────────────────────────┐
│ python app.py        │ ─────────────► │ bash -s  (streamed)      │
│ browser 127.0.0.1:8765│  stdin push    │ tmux acc_burn (gpu-burn) │
│ data/ results+report │ ◄───────────── │ tmux acc_mon  (30s poll) │
└──────────────────────┘  stdout back   └──────────────────────────┘
```

## Quick start

```bash
cd gpu-acceptance
python app.py        # or run.bat on Windows; opens http://127.0.0.1:8765
```

1. Local requirements: Python 3.8+ (stdlib only), an OpenSSH client (bundled with Windows).
2. Server requirements: root key-based SSH login and tmux; burn-in / P2P build additionally
   needs git + nvcc (CUDA toolkit) + make.
3. Connection settings are filled in the header of the UI (host / user / port / key path /
   remote Python / load-test URL), or preseeded via `GPU_ACC_HOST`, `GPU_ACC_USER`,
   `GPU_ACC_PORT`, `GPU_ACC_KEY`, `GPU_ACC_PY`, `GPU_ACC_URL`; they persist in `data/conn.json`.
4. **Credential safety**: the SSH private key stays in your local `~/.ssh/`; no usable
   credential literal is written into source or config. The web UI binds to `127.0.0.1` only.

## Acceptance items → UI buttons → pass criteria

| Item | UI section | Script | Auto-verdict |
|---|---|---|---|
| ① Unboxing check | ① Hardware inventory | `01_hardware_inventory.sh` | GPU model/SN/UUID table + chassis SN for contract cross-check; appearance ticked in "manual confirmations" |
| ② Config check | ② Config collection & compare | `02_config_check.sh` | GPU count/model, CPU cores, RAM, data-disk size vs acceptance criteria (editable thresholds) |
| ③ 72h burn-in | ③ Preflight → Start → Status → Stop | `00–03_*.sh` | Progress, peak temp/power, ECC counters, Xid; ECC must be 0 |
| ④ P2P bandwidth (mandatory) | ④ P2P test | `04_p2p_bandwidth.sh` | nvidia-smi topo + p2pBandwidthLatencyTest matrix (peak auto-extracted, human review) |
| ④ Memory BW / compute | ④ Performance | `04_perf_bench.py` | D2D bandwidth and FP32/TF32/FP16/FP8 TFLOPS vs ratio thresholds (editable) |
| ④ Concurrency load test | ④ Load test | `04_concurrency_test.py` (local) | P50/P95/P99 and error rate vs thresholds; intranet targets need the explicit "intranet" checkbox |
| ⑤ Software acceptance | ⑤ Software verification | `05_software_check.sh` | Driver/CUDA/PyTorch CUDA usable (real matmul)/TF/image list |
| ⑥ Documentation | ⑥ Manual confirm + report | local | Aggregates auto verdicts + manual confirmations into `data/acceptance report *.md` |

**Note**: the performance check (`04_perf_bench.py`) requires PyTorch — set "remote Python" in
the header to an interpreter that has it, e.g. `/root/miniconda3/envs/xxx/bin/python`
(conda env list from check ⑤ helps).

## 72h burn-in workflow

1. `③ Preflight`: verify tmux/git/nvcc and idle GPUs.
2. `③ Start burn-in`: locates or builds gpu-burn, runs it in `tmux acc_burn` (default 4320 min
   = 72h, editable), monitors at 30 s intervals into `/root/acceptance/burnin_monitor.csv`.
   **Disconnecting SSH or closing the console does not interrupt it.**
3. Any time, `③ Burn-in status` (optional 30 s auto refresh): progress, peak temp/power,
   ECC counters, Xid kernel log.
4. After 72h: `③ Stop burn-in`, then run `③ ECC check` (must be 0 errors).

## Default criteria (preset for an 8×L40 machine, editable per contract)

| Metric | Spec | Pass line |
|---|---|---|
| FP32 | 90.5 TFLOPS | ≥95% |
| TF32 / FP16 / FP8 | 90.5 / 181 / 362 TFLOPS | ≥95% (FP16/TF32 informative) |
| Memory bandwidth | 864 GB/s | ≥85% (D2D copy counts read+write; 85–92% is normal) |
| P2P bandwidth | PCIe Gen4 x16 (no-NVLink SKUs) | typically 20–26 GB/s; default floor 20 |
| ECC | 0 correctable / 0 uncorrectable | =0 (mandatory) |

## Manual command cheat sheet (no UI)

```bash
H=root@<server-ip>               # key-based login (replace with your target)
K=~/.ssh/id_ed25519              # local private key

# ① Hardware inventory
ssh -i $K $H 'nvidia-smi --query-gpu=index,name,serial,uuid,memory.total,driver_version --format=csv'
# ② Config check
ssh -i $K $H 'nproc; free -g | head -2; df -h /data2; nvidia-smi --list-gpus | wc -l'
# ③ Burn-in (scripts pushed via stdin; workload survives disconnects inside tmux)
ssh -i $K $H 'bash -s' 4320 < scripts/03_burnin_start.sh
ssh -i $K $H 'bash -s'       < scripts/03_burnin_status.sh
ssh -i $K $H 'bash -s'       < scripts/03_burnin_stop.sh
ssh -i $K $H 'bash -s'       < scripts/03_ecc_check.sh
# Watch burn-in: ssh -i $K $H 'tmux attach -t acc_burn'  /  tail -f /root/acceptance/burnin.log
# ④ P2P / performance
ssh -i $K $H 'bash -s'       < scripts/04_p2p_bandwidth.sh
ssh -i $K $H '/root/miniconda3/envs/xxx/bin/python -' < scripts/04_perf_bench.py
# ④ Concurrency load test (local)
python scripts/04_concurrency_test.py http://<server-ip>:8000/v1/completions 120 10 \
  '{"model":"model-name","prompt":"hello","max_tokens":16}'
# ⑤ Software acceptance (ACC_PY selects the interpreter to verify)
ssh -i $K $H 'env ACC_PY=python3 bash -s' < scripts/05_software_check.sh
```

## Results & archiving

- `data/results.json`: all check results (metrics + verdicts + timestamps).
- `data/logs/*.log`: full raw output of every run.
- `data/acceptance report *.md`: one-click report (GPU serial-number register, per-item
  verdicts, conclusion, signature block). Archive it together with the factory test report
  number and warranty voucher number to complete item ⑥.

## Troubleshooting

- **GitHub clone fails** (gpu-burn / cuda-samples): download elsewhere and `scp` to the server
  as `/root/gpu-burn` (prebuilt `gpu_burn` binary in `/root/gpu-burn/`) or `/root/cuda-samples`;
  the start script reuses them.
- **Performance check fails to import torch**: wrong remote interpreter — use the full path of
  the conda env's python.
- **FP8 fails**: torch < 2.1 or no `float8_e4m3fn`; the item is marked "not judged" — supplement
  with gpu-burn/TensorRT.
- **Load-test target unreachable**: verify the service URL first; vLLM defaults to
  `:8000/v1/completions` (POST) or `:8000/v1/models` (GET).
- **Script line endings on Windows**: the console normalizes CRLF to LF when pushing, so editing
  scripts locally is safe.
