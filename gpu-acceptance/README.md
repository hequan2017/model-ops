# GPU 服务器验收控制台（本地 Web + SSH 远程执行）

[中文](README.md) | [English](README.en.md)

GPU 服务器到货验收工具。**控制台只在本机运行**，所有验收脚本通过 SSH 推送到目标服务器以
`bash -s` / `<python> -` 方式执行，**服务器上不部署任何界面、服务或常驻程序**（烤机负载运行在
服务器 tmux 会话中，这是压测本身的必然要求）。目标服务器地址不内置，首次使用在界面顶部填写，
或用环境变量提供。

```
本机 (Windows / Linux / macOS)          目标 GPU 服务器
┌──────────────────────┐   ssh -i key   ┌──────────────────────────┐
│ python app.py        │ ─────────────► │ bash -s  (验收脚本流式执行)│
│ 浏览器 127.0.0.1:8765 │   stdin 推送    │ tmux acc_burn (gpu-burn)  │
│ data/ 结果与报告归档   │ ◄───────────── │ tmux acc_mon  (30s 采样)   │
└──────────────────────┘   stdout 回传   └──────────────────────────┘
```

## 快速开始

```bash
cd gpu-acceptance
python app.py        # 或双击 run.bat；自动打开 http://127.0.0.1:8765
```

1. 环境要求（本机）：Python 3.8+（零第三方依赖）、OpenSSH 客户端（Windows 自带）。
2. 环境要求（服务器）：root 密钥登录、tmux；烤机/P2P 构建另需 git + nvcc(CUDA 工具链) + make。
3. 连接参数在界面顶部填写（主机/用户/端口/私钥/远端 Python/压测目标），或用环境变量
   `GPU_ACC_HOST` / `GPU_ACC_USER` / `GPU_ACC_PORT` / `GPU_ACC_KEY` / `GPU_ACC_PY` / `GPU_ACC_URL`
   预置，保存到 `data/conn.json` 后以文件为准。
4. **凭据安全**：SSH 私钥保留在本机 `~/.ssh/`，源码与配置不写入任何可用凭据字面量；
   Web 仅监听 `127.0.0.1`，外部无法访问。

## 验收项 → 界面按钮 → 判定标准对照

| 验收事项 | 界面位置 | 执行脚本 | 自动判定 |
|---|---|---|---|
| ① 开箱核对 | ① 硬件清点 | `01_hardware_inventory.sh` | 输出 GPU 序列号/UUID/整机 SN 表，与合同逐项人工比对；外观在“人工确认”打勾 |
| ② 配置核对 | ② 配置采集与比对 | `02_config_check.sh` | GPU 数量/型号、CPU 核数、内存、数据盘容量 vs 验收标准（阈值可改） |
| ③ 72h 烤机 | ③ 环境预检→启动→状态→停止 | `00~03_*.sh` | 进度/温度功耗峰值/ECC 计数/Xid；ECC 必须 0 错误 |
| ④ P2P 带宽（强制项） | ④ P2P 测试 | `04_p2p_bandwidth.sh` | nvidia-smi topo + p2pBandwidthLatencyTest 矩阵（自动提取峰值，人工复核） |
| ④ 显存带宽/算力 | ④ 性能实测 | `04_perf_bench.py` | D2D 带宽与 FP32/TF32/FP16/FP8 TFLOPS 按达标率判定（阈值可改） |
| ④ 并发压测 | ④ 并发压测 | `04_concurrency_test.py`（本地） | P50/P95/P99、错误率按阈值判定；内网目标需勾选“内网目标” |
| ⑤ 软件验收 | ⑤ 软件环境验证 | `05_software_check.sh` | 驱动/CUDA/PyTorch CUDA 可用（含真实矩阵运算）/TF/镜像列表 |
| ⑥ 文档归档 | ⑥ 人工确认 + 生成报告 | 本地 | 汇总自动判定+人工确认，输出 `data/验收报告_*.md` |

**重要**：性能实测（`04_perf_bench.py`）走 PyTorch，请先在顶部“远端 Python”填入装有 PyTorch 的
解释器，例如 `/root/miniconda3/envs/xxx/bin/python`（可用 `⑤ 软件验证` 的 conda env list 辅助确认）。

## 72h 烤机操作流程

1. `③ 环境预检`：确认 tmux/git/nvcc 齐全、GPU 空闲。
2. `③ 启动烤机`：自动定位或克隆编译 gpu-burn，负载跑在 `tmux acc_burn`（默认 4320 分钟 = 72h，
   标准里可改），监控 30 秒采样到服务器 `/root/acceptance/burnin_monitor.csv`。
   **断开 SSH / 关掉本控制台均不影响**。
3. 期间随时点 `③ 烤机状态`（可勾选 30s 自动刷新）：看进度、峰值温度/功耗、ECC 计数、Xid 日志。
4. 72h 结束后 `③ 停止烤机`，再跑一次 `③ ECC 日志核查`（要求 0 错误）。

## 验收标准默认值（按 L40 ×8 整机预置，界面可按合同修改）

| 指标 | 标称 | 达标线 |
|---|---|---|
| FP32 | 90.5 TFLOPS | ≥95% |
| TF32 / FP16 / FP8 | 90.5 / 181 / 362 TFLOPS | ≥95%（FP16/TF32 为参考项） |
| 显存带宽 | 864 GB/s | ≥85%（D2D 拷贝含读写，85%~92% 为正常水平） |
| P2P 带宽 | PCIe Gen4 x16（无 NVLink 机型） | 实测一般 20~26 GB/s，默认下限 20 |
| ECC | 0 可纠正 / 0 不可纠正 | =0（强制） |

## 手工命令速查（不开界面、直接在终端执行）

```bash
H=root@<服务器IP>                # 密钥登录（目标地址自行替换）
K=~/.ssh/id_ed25519             # 本机私钥

# ① 硬件清点
ssh -i $K $H 'nvidia-smi --query-gpu=index,name,serial,uuid,memory.total,driver_version --format=csv'
# ② 配置核对
ssh -i $K $H 'nproc; free -g | head -2; df -h /data2; nvidia-smi --list-gpus | wc -l'
# ③ 烤机（推送脚本执行；负载在 tmux 中，断线不中断）
ssh -i $K $H 'bash -s' 4320 < scripts/03_burnin_start.sh
ssh -i $K $H 'bash -s'       < scripts/03_burnin_status.sh
ssh -i $K $H 'bash -s'       < scripts/03_burnin_stop.sh
ssh -i $K $H 'bash -s'       < scripts/03_ecc_check.sh
# 手工看烤机: ssh -i $K $H 'tmux attach -t acc_burn'  /  tail -f /root/acceptance/burnin.log
# ④ P2P / 性能
ssh -i $K $H 'bash -s'       < scripts/04_p2p_bandwidth.sh
ssh -i $K $H '/root/miniconda3/envs/xxx/bin/python -' < scripts/04_perf_bench.py
# ④ 并发压测（本地）
python scripts/04_concurrency_test.py http://<服务器IP>:8000/v1/completions 120 10 \
  '{"model":"模型名","prompt":"你好","max_tokens":16}'
# ⑤ 软件验收（ACC_PY 指定待验证解释器）
ssh -i $K $H 'env ACC_PY=python3 bash -s' < scripts/05_software_check.sh
```

## 结果与归档

- `data/results.json`：全部检查结果（指标 + 判定 + 时间戳）。
- `data/logs/*.log`：每次执行的完整原始输出。
- `data/验收报告_*.md`：一键生成的验收报告（含 GPU 序列号登记表、逐项判定、结论、签字栏），
  连同出厂测试报告编号、质保凭证编号一起归档即可完成第⑥项。

## 常见问题

- **GitHub 克隆失败**（gpu-burn / cuda-samples）：可在有网的机器下载后 `scp` 到服务器
  `/root/gpu-burn`（已编译好的 `gpu_burn` 放 `/root/gpu-burn/`）或 `/root/cuda-samples`，再点启动即可复用。
- **性能实测报 import torch 失败**：远端 Python 解释器不对，改用 conda 环境内的 python 全路径。
- **FP8 实测失败**：torch 版本低于 2.1 或无 `float8_e4m3fn`；该项会标记“未判定”，可用 gpu-burn/TensorRT 补充。
- **压测目标打不通**：先在浏览器/curl 确认服务地址；vLLM 默认 `:8000/v1/completions`（POST）或 `:8000/v1/models`（GET）。
- **Windows 下脚本行尾**：控制台推送时自动把 CRLF 归一为 LF，直接编辑脚本不影响远端执行。
