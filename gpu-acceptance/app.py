#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
GPU 服务器验收控制台 —— 本地 Web 界面 + SSH 远程执行
====================================================
- 本程序只在本机运行（默认仅监听 127.0.0.1:8765）；所有验收脚本通过 SSH 推送到
  目标服务器以 `bash -s` / `<python> -` 方式执行，远端不部署任何常驻服务或界面。
- 目标服务器地址不内置：首次使用在界面顶部填写主机/用户/私钥，或用环境变量
  GPU_ACC_HOST / GPU_ACC_USER / GPU_ACC_PORT / GPU_ACC_KEY / GPU_ACC_PY / GPU_ACC_URL
  提供（保存在 data/conn.json，之后以该文件为准）。
- 凭据约定：SSH 私钥保留在本机（默认 ~/.ssh/id_ed25519 或 id_rsa）；源码与配置中
  不写入任何可用凭据字面量。
- 结果与报告保存在 data/ 目录（results.json、logs/、验收报告_*.md）。

启动: python app.py   （或 run.bat / run.sh）
"""
import json
import os
import shlex
import shutil
import subprocess
import sys
import threading
import time
import uuid
import webbrowser
from datetime import datetime
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path
from urllib.parse import parse_qs, quote, urlparse

BASE = Path(__file__).resolve().parent
WEB = BASE / "web"
DATA = BASE / "data"
LOGS = DATA / "logs"
for d in (DATA, LOGS):
    d.mkdir(exist_ok=True)
RESULTS_FILE = DATA / "results.json"
CONN_FILE = DATA / "conn.json"
SPEC_FILE = DATA / "spec.json"

HTTP_PORT = int(os.environ.get("GPU_ACC_WEB_PORT", "8765"))
BIND = "127.0.0.1"


# --------------------------------------------------------------------------
# 基础工具
# --------------------------------------------------------------------------
def find_ssh():
    p = shutil.which("ssh")
    if p:
        return p
    cand = Path(os.environ.get("SystemRoot", r"C:\Windows")) / "System32" / "OpenSSH" / "ssh.exe"
    return str(cand) if cand.exists() else "ssh"


SSH_BIN = find_ssh()


def load_json(path, default):
    try:
        return json.loads(Path(path).read_text(encoding="utf-8"))
    except Exception:  # noqa: BLE001
        return default


def save_json(path, obj):
    Path(path).write_text(json.dumps(obj, ensure_ascii=False, indent=2), encoding="utf-8")


RES_LOCK = threading.RLock()


def load_results():
    with RES_LOCK:
        return load_json(RESULTS_FILE, {})


def save_results(r):
    with RES_LOCK:
        save_json(RESULTS_FILE, r)


def default_conn():
    key = os.environ.get("GPU_ACC_KEY", "")
    if not key:
        for k in ("id_ed25519", "id_rsa"):
            p = Path.home() / ".ssh" / k
            if p.exists():
                key = str(p)
                break
    return {
        "host": os.environ.get("GPU_ACC_HOST", ""),
        "user": os.environ.get("GPU_ACC_USER", ""),
        "port": int(os.environ.get("GPU_ACC_PORT", "22")),
        "key": key,
        "remote_python": os.environ.get("GPU_ACC_PY", "python3"),
        "service_url": os.environ.get("GPU_ACC_URL", ""),
    }


def load_conn():
    c = default_conn()
    saved = load_json(CONN_FILE, {})
    if isinstance(saved, dict):
        c.update({k: v for k, v in saved.items() if k in c})
    return c


# 验收标准（默认按 L40 ×8 整机设定，界面可改）
DEFAULT_SPEC = {
    "gpu_count": 8,
    "gpu_model_contains": "L40",
    "logical_cores": 96,
    "mem_gb": 503,
    "mem_tol_pct": 5,
    "data_disk_mount": "/data2",
    "data_disk_min_gb": 3400,
    "burn_duration_min": 4320,
    "gpu_temp_max": 85,
    "p2p_min_gbps": 20,
    "mem_bw_gbps": 864,
    "mem_bw_min_ratio": 0.85,
    "fp32_tflops": 90.5,
    "tf32_tflops": 90.5,
    "fp16_tflops": 181,
    "fp8_tflops": 362,
    "flops_min_ratio": 0.95,
    "conc_users": 10,
    "conc_duration_s": 120,
    "p95_max_ms": 3000,
    "err_max": 0,
}


def load_spec():
    s = dict(DEFAULT_SPEC)
    saved = load_json(SPEC_FILE, {})
    if isinstance(saved, dict):
        s.update({k: v for k, v in saved.items() if k in s})
    return s


# --------------------------------------------------------------------------
# 检查项定义（供界面渲染与执行分发）
# --------------------------------------------------------------------------
CHECKS = [
    {"id": "hw", "cat": "① 开箱核对", "name": "硬件清点",
     "desc": "GPU 型号/序列号/UUID、整机 SN、主板/BIOS、网卡、NVMe 清单（序列号供合同比对与注册）",
     "btn": "运行硬件清点",
     "rows_header": ["GPU", "型号", "序列号", "UUID", "PCI 总线", "显存", "驱动"]},
    {"id": "cfg", "cat": "② 配置核对", "name": "配置采集与比对",
     "desc": "CPU 核数 / 内存容量 / 磁盘容量 / GPU 数量与型号，自动与下方验收标准比对", "btn": "运行配置核对"},
    {"id": "preflight", "cat": "③ 72h 满载烤机", "name": "环境预检",
     "desc": "烤机前置依赖：tmux/git/nvcc/gcc、GPU 空闲状态、磁盘空间", "btn": "运行预检"},
    {"id": "burn_start", "cat": "③ 72h 满载烤机", "name": "启动烤机",
     "desc": "gpu-burn 满载（时长见验收标准）+ 30 秒采样监控，运行于 tmux 后台，断开 SSH 不中断", "btn": "启动烤机"},
    {"id": "burn_status", "cat": "③ 72h 满载烤机", "name": "烤机状态",
     "desc": "进度、温度/功耗峰值、ECC 计数、Xid 内核日志（可开启 30 秒自动刷新）", "btn": "刷新状态", "auto": True},
    {"id": "burn_stop", "cat": "③ 72h 满载烤机", "name": "停止烤机",
     "desc": "结束负载与监控会话，日志保留在服务器 /root/acceptance/", "btn": "停止烤机"},
    {"id": "ecc", "cat": "③ 72h 满载烤机", "name": "ECC 日志核查",
     "desc": "ECC 模式、可纠正/不可纠正错误计数（验收要求 0 错误）、内核日志",
     "btn": "核查 ECC",
     "rows_header": ["GPU", "ECC 模式", "可纠正(易失)", "不可纠正(易失)", "可纠正(累计)", "不可纠正(累计)"]},
    {"id": "p2p", "cat": "④ 性能实测", "name": "GPU P2P 带宽（强制验收项）",
     "desc": "nvidia-smi topo + p2pBandwidthLatencyTest（首次自动从 cuda-samples 稀疏克隆构建，需 git+nvcc+make）",
     "btn": "运行 P2P 测试"},
    {"id": "perf", "cat": "④ 性能实测", "name": "显存带宽与算力实测",
     "desc": "D2D 拷贝带宽、FP32/TF32/FP16/FP8 matmul TFLOPS，与标称值自动比对（阈值见验收标准；解释器在顶部连接设置中配置）",
     "btn": "运行性能实测"},
    {"id": "conc", "cat": "④ 性能实测", "name": "10 人并发压测",
     "desc": "从本机模拟师生并发访问推理/教学服务，统计 P50/P95/P99 与错误率（压测内网目标请勾选“内网目标”）",
     "btn": "开始压测", "inputs": True},
    {"id": "sw", "cat": "⑤ 软件验收", "name": "软件环境验证",
     "desc": "驱动/CUDA/nvcc/Docker 与课程镜像、PyTorch 与 TensorFlow GPU 可用性（含真实 GPU 矩阵运算）",
     "btn": "运行软件验证"},
]
CATS = ["① 开箱核对", "② 配置核对", "③ 72h 满载烤机", "④ 性能实测", "⑤ 软件验收"]
RUNNABLE = {c["id"] for c in CHECKS}


# --------------------------------------------------------------------------
# 任务系统
# --------------------------------------------------------------------------
TASKS = {}
TLOCK = threading.RLock()


def new_task(check_id):
    tid = uuid.uuid4().hex[:8]
    task = {"id": tid, "check": check_id, "status": "running",
            "output": [], "metrics": {}, "started": time.time()}
    with TLOCK:
        TASKS[tid] = task
    return task


def emit(task, line):
    with TLOCK:
        task["output"].append(line)


def parse_metric(task, line):
    body = line[2:]
    if body.startswith("GPU|") or body.startswith("ECC|"):
        task["metrics"].setdefault("rows", []).append(body.split("|"))
        return
    if "=" in body:
        k, v = body.split("=", 1)
        task["metrics"][k.strip()] = v.strip()


def ssh_argv(conn, remote_cmd=None):
    argv = [SSH_BIN, "-p", str(conn.get("port", 22)),
            "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=accept-new",
            "-o", "ConnectTimeout=10", "-o", "ServerAliveInterval=30"]
    if conn.get("key"):
        argv += ["-i", str(Path(conn["key"]).expanduser())]
    argv += [f"{conn['user']}@{conn['host']}"]
    if remote_cmd:
        argv += remote_cmd
    return argv


def stream_proc(task, argv, stdin_file=None):
    """启动子进程并把输出逐行送入任务；@@ 开头的行解析为指标。"""
    fh = open(stdin_file, "rb") if stdin_file else None
    try:
        proc = subprocess.Popen(
            argv, stdin=fh or subprocess.DEVNULL,
            stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
            text=True, encoding="utf-8", errors="replace")
        task["_proc"] = proc
        for line in proc.stdout:
            line = line.rstrip("\n").rstrip("\r")
            if line.startswith("@@"):
                parse_metric(task, line)
            emit(task, line)
        rc = proc.wait()
        task.pop("_proc", None)
        return rc
    finally:
        if fh:
            fh.close()


def script_bytes(name):
    """读取脚本并统一为 LF 行尾（Windows 下编辑也不影响远端 bash）。"""
    return (BASE / "scripts" / name).read_bytes().replace(b"\r\n", b"\n")


def run_remote_script(task, conn, script, args=(), env_prefix=""):
    staged = DATA / f".stage_{task['id']}_{script}"
    staged.write_bytes(script_bytes(script))
    cmd = env_prefix + "bash -s" + ("".join(" " + shlex.quote(a) for a in args))
    argv = ssh_argv(conn, [cmd])
    rc = stream_proc(task, argv, stdin_file=staged)
    try:
        staged.unlink()
    except OSError:
        pass
    return rc


def run_remote_python(task, conn, script):
    staged = DATA / f".stage_{task['id']}_{script}"
    staged.write_bytes(script_bytes(script))
    argv = ssh_argv(conn, [conn["remote_python"] + " -"])
    rc = stream_proc(task, argv, stdin_file=staged)
    try:
        staged.unlink()
    except OSError:
        pass
    return rc


def run_local(task, script, args):
    staged = DATA / f".stage_{task['id']}_{script}"
    staged.write_bytes(script_bytes(script))
    argv = [sys.executable, str(staged)] + [str(a) for a in args]
    rc = stream_proc(task, argv)
    try:
        staged.unlink()
    except OSError:
        pass
    return rc


def dispatch(check_id, params):
    conn = load_conn()
    spec = load_spec()
    task = new_task(check_id)

    def work():
        try:
            if check_id == "hw":
                rc = run_remote_script(task, conn, "01_hardware_inventory.sh")
            elif check_id == "cfg":
                rc = run_remote_script(task, conn, "02_config_check.sh")
            elif check_id == "preflight":
                rc = run_remote_script(task, conn, "00_preflight.sh")
            elif check_id == "burn_start":
                rc = run_remote_script(task, conn, "03_burnin_start.sh",
                                       args=(str(spec.get("burn_duration_min", 4320)),))
            elif check_id == "burn_status":
                rc = run_remote_script(task, conn, "03_burnin_status.sh")
            elif check_id == "burn_stop":
                rc = run_remote_script(task, conn, "03_burnin_stop.sh")
            elif check_id == "ecc":
                rc = run_remote_script(task, conn, "03_ecc_check.sh")
            elif check_id == "p2p":
                rc = run_remote_script(task, conn, "04_p2p_bandwidth.sh")
            elif check_id == "perf":
                rc = run_remote_python(task, conn, "04_perf_bench.py")
            elif check_id == "sw":
                rc = run_remote_script(task, conn, "05_software_check.sh",
                                       env_prefix="env ACC_PY=%s " % shlex.quote(conn["remote_python"]))
            elif check_id == "conc":
                url = (params.get("url") or conn.get("service_url") or "").strip()
                if not url:
                    emit(task, "[错误] 未配置压测目标 URL")
                    task["status"] = "error"
                    return
                args = [url, str(params.get("duration", spec.get("conc_duration_s", 120))),
                        str(params.get("users", spec.get("conc_users", 10)))]
                body = (params.get("body") or "").strip()
                if body:
                    args.append(body)
                if params.get("allow_private"):
                    args.append("--allow-private")
                rc = run_local(task, "04_concurrency_test.py", args)
            else:
                emit(task, f"[错误] 未知检查项: {check_id}")
                rc = 1
            if task.get("status") != "stopped":
                task["status"] = "done" if rc == 0 else "error"
                if rc != 0:
                    task["metrics"]["EXIT_CODE"] = str(rc)
        except Exception as e:  # noqa: BLE001
            emit(task, f"[本地异常] {type(e).__name__}: {e}")
            if task.get("status") != "stopped":
                task["status"] = "error"
        finally:
            save_result(task)

    threading.Thread(target=work, daemon=True).start()
    return task


# --------------------------------------------------------------------------
# 判定逻辑（指标 vs 验收标准）
# --------------------------------------------------------------------------
def _n(v):
    try:
        return float(str(v).strip())
    except (TypeError, ValueError):
        return None


def judge(check_id, m, spec):
    rows = []

    def add(label, actual, expected, ok):
        rows.append({"label": label, "actual": "" if actual in (None, "") else str(actual),
                     "expected": str(expected), "pass": ok})

    n = _n
    if check_id == "cfg":
        add("GPU 数量", m.get("GPU_COUNT"), spec["gpu_count"],
            n(m.get("GPU_COUNT")) == n(spec["gpu_count"]))
        model = str(m.get("GPU_MODEL", ""))
        add("GPU 型号", m.get("GPU_MODEL"), f"包含 {spec['gpu_model_contains']}",
            (spec["gpu_model_contains"].lower() in model.lower()) if model else None)
        add("CPU 逻辑核数", m.get("LOGICAL_CPUS"), spec["logical_cores"],
            n(m.get("LOGICAL_CPUS")) == n(spec["logical_cores"]))
        mg, eg = n(m.get("MEM_TOTAL_GB")), n(spec["mem_gb"])
        tol = n(spec["mem_tol_pct"]) / 100.0
        add("内存容量 GB (free 可用总量)", m.get("MEM_TOTAL_GB"),
            f"{spec['mem_gb']} ±{spec['mem_tol_pct']}%",
            (abs(mg - eg) <= eg * tol) if mg is not None else None)
        mount = str(spec["data_disk_mount"])
        key = "DISK_" + mount.replace("/", "_") + "_GB"
        dv = n(m.get(key))
        add(f"数据盘 {mount} 容量 GB", m.get(key), f"≥{spec['data_disk_min_gb']}",
            (dv >= n(spec["data_disk_min_gb"])) if dv is not None else None)
    elif check_id == "preflight":
        for k in sorted(m):
            if k.startswith("CMD_"):
                cmd = k[4:]
                add(f"命令 {cmd}", m.get(k), "OK", m.get(k) == "OK")
    elif check_id == "perf":
        mb, mb_e = n(m.get("MEM_BW_GBPS")), n(spec["mem_bw_gbps"])
        r = n(spec["mem_bw_min_ratio"])
        add("显存带宽 GB/s（D2D 拷贝，读+写）", m.get("MEM_BW_GBPS"),
            f"≥{mb_e}×{r}={mb_e * r:.0f}", (mb >= mb_e * r) if mb is not None else None)
        for skey, mkey, label in (("fp32_tflops", "FP32_TFLOPS", "FP32 算力 TFLOPS"),
                                  ("tf32_tflops", "TF32_TFLOPS", "TF32 算力 TFLOPS（参考）"),
                                  ("fp16_tflops", "FP16_TFLOPS", "FP16 算力 TFLOPS（参考）"),
                                  ("fp8_tflops", "FP8_TFLOPS", "FP8 算力 TFLOPS")):
            av, ex, rr = n(m.get(mkey)), n(spec[skey]), n(spec["flops_min_ratio"])
            add(label, m.get(mkey), f"≥{ex}×{rr}={ex * rr:.1f}",
                (av >= ex * rr) if (av is not None and "FAILED" not in str(m.get(mkey))) else None)
    elif check_id == "p2p":
        v, e = n(m.get("P2P_MAX_GBPS")), n(spec["p2p_min_gbps"])
        add("P2P/总线带宽峰值 GB/s（自动提取，矩阵需人工复核）", m.get("P2P_MAX_GBPS"),
            f"≥{e}", (v >= e) if v is not None else None)
    elif check_id == "burn_status":
        if m.get("RUNNING") == "never":
            add("状态", "未启动烤机", "-", None)
        else:
            add("负载运行中", m.get("RUNNING"), "yes", str(m.get("RUNNING")) == "yes")
            el, du = n(m.get("ELAPSED_MIN")), n(m.get("DURATION_MIN"))
            if el is not None and du:
                add("已运行 / 计划（分钟）", f"{m.get('ELAPSED_MIN')} / {m.get('DURATION_MIN')}",
                    f"≥{spec['burn_duration_min']}", el >= n(spec["burn_duration_min"]))
            for k in ("ECC_CORR_TOTAL", "ECC_UNCORR_TOTAL"):
                v = n(m.get(k))
                add(k, m.get(k), "=0", (v == 0) if v is not None else None)
            tmax = None
            for k, v in m.items():
                if k.endswith("_MAX_TEMP"):
                    tv = n(v)
                    if tv is not None and (tmax is None or tv > tmax):
                        tmax = tv
            if tmax is not None:
                add("GPU 峰值温度 ℃", f"{tmax:.0f}", f"≤{spec['gpu_temp_max']}",
                    tmax <= n(spec["gpu_temp_max"]))
    elif check_id == "burn_start":
        add("烤机启动", m.get("STARTED"), "yes", m.get("STARTED") == "yes")
    elif check_id == "burn_stop":
        add("烤机停止", m.get("STOPPED"), "yes", m.get("STOPPED") == "yes")
    elif check_id == "ecc":
        c, u = n(m.get("ECC_CORR_TOTAL")), n(m.get("ECC_UNCORR_TOTAL"))
        add("ECC 可纠正错误（累计合计）", m.get("ECC_CORR_TOTAL"), "=0", (c == 0) if c is not None else None)
        add("ECC 不可纠正错误（累计合计）", m.get("ECC_UNCORR_TOTAL"), "=0", (u == 0) if u is not None else None)
    elif check_id == "sw":
        add("NVIDIA 驱动", m.get("DRIVER"), "已安装", bool(m.get("DRIVER")))
        add("CUDA 运行时", m.get("CUDA"), "已安装", bool(m.get("CUDA")))
        add("PyTorch", m.get("TORCH"), "可导入且 CUDA 可用", str(m.get("TORCH_CUDA_OK")) == "true")
        tf = str(m.get("TF", ""))
        add("TensorFlow", m.get("TF"), "可导入（课程需要时）",
            None if (not tf or tf.startswith("FAILED")) else True)
    elif check_id == "conc":
        v, e = n(m.get("P95_MS")), n(spec["p95_max_ms"])
        add("P95 响应 ms", m.get("P95_MS"), f"≤{e}", (v <= e) if v is not None else None)
        er, em = n(m.get("ERRORS")), n(spec.get("err_max", 0))
        add("错误请求数", m.get("ERRORS"), f"≤{em}", (er <= em) if er is not None else None)
    return rows


def save_result(task):
    rid = task["check"]
    log_rel = None
    try:
        name = f"{rid}_{task['id']}.log"
        (LOGS / name).write_text("\n".join(task["output"]), encoding="utf-8")
        log_rel = "logs/" + name
    except OSError:
        pass
    rec = {"ts": datetime.now().strftime("%Y-%m-%d %H:%M:%S"), "status": task["status"],
           "metrics": dict(task["metrics"]), "log": log_rel}
    rec["judge"] = judge(rid, rec["metrics"], load_spec())
    results = load_results()
    results[rid] = rec
    save_results(results)


# --------------------------------------------------------------------------
# 报告生成（⑥ 文档归档）
# --------------------------------------------------------------------------
REQUIRED_MANUAL = [
    ("appearance_ok", "① 开箱外观无运输损伤"),
    ("course_image_ok", "⑤ 课程镜像已部署验证"),
    ("accounts_ok", "⑤ 师生账号已开通验证"),
    ("sn_registered", "⑥ GPU 序列号已注册"),
    ("warranty_archived", "⑥ 质保凭证已归档"),
]


def build_report():
    conn, spec, res = load_conn(), load_spec(), load_results()
    man = res.get("manual", {})
    problems = []
    L = []
    ap = L.append
    ap("# GPU 服务器验收报告")
    ap("")
    ap(f"- 验收对象: `{conn['user']}@{conn['host']}:{conn['port']}`")
    ap(f"- 整机标称: {spec['gpu_count']} × {spec['gpu_model_contains']}，{spec['logical_cores']} 逻辑核，"
       f"{spec['mem_gb']}GB 内存")
    ap(f"- 报告生成: {datetime.now():%Y-%m-%d %H:%M:%S}")
    ap("")

    def judge_table(rid):
        rec = res.get(rid)
        if not rec:
            ap(f"> 该项尚未执行（检查项 `{rid}`）"); ap("")
            return
        rows = judge(rid, rec.get("metrics", {}), spec)
        if rows:
            ap("| 判定项 | 实测 | 标准 | 结果 |")
            ap("|---|---|---|---|")
            for r in rows:
                mark = {True: "✅", False: "❌", None: "—"}[r["pass"]]
                ap(f"| {r['label']} | {r['actual']} | {r['expected']} | {mark} |")
                if r["pass"] is False:
                    problems.append(f"{rid}: {r['label']} 实测 {r['actual']}，标准 {r['expected']}")
        if rec.get("log"):
            ap(f"\n原始输出: `{rec['log']}`")
        ap("")

    ap("## 一、开箱核对")
    hw = res.get("hw", {}).get("metrics", {})
    if hw.get("rows"):
        ap("| # | 型号 | 序列号 | UUID | PCI 总线 | 显存 | 驱动 |")
        ap("|---|---|---|---|---|---|---|")
        for r in hw["rows"]:
            cells = [c.strip() or "N/A" for c in r[1:8]]
            while len(cells) < 7:
                cells.append("")
            ap("| " + " | ".join(cells) + " |")
        ap("")
    else:
        ap("> 硬件清点未执行"); ap("")
    if hw.get("SYSTEM_SN"):
        ap(f"整机序列号: `{hw['SYSTEM_SN']}`（用于合同逐项比对与序列号注册）")
    ap(f"外观无运输损伤: {'✅ 已确认' if man.get('appearance_ok') else '❌ 未确认'}"
       f"　合同编号: {man.get('contract_no') or '—'}")
    ap("")

    ap("## 二、配置核对（与配置单逐项比对）")
    judge_table("cfg")

    ap("## 三、72h 满载烤机")
    judge_table("burn_status")
    bs = res.get("burn_status", {}).get("metrics", {})
    if bs:
        keep = ("ELAPSED_MIN", "DURATION_MIN", "PROGRESS", "ECC_CORR_TOTAL", "ECC_UNCORR_TOTAL", "MON_SAMPLES")
        for k in keep:
            if k in bs:
                ap(f"- {k}: {bs[k]}")
        for k, v in bs.items():
            if k.endswith("_MAX_TEMP") or k.endswith("_MAX_POWER"):
                ap(f"- {k}: {v}")
        ap("")
    ap("### ECC 日志核查（要求 0 错误）")
    judge_table("ecc")

    ap("## 四、性能实测（偏差 ≤5%，阈值可在验收标准中调整）")
    ap("### GPU P2P 带宽（强制验收项）")
    judge_table("p2p")
    ap("### 显存带宽与算力")
    judge_table("perf")
    ap("### 并发压测（模拟师生访问）")
    judge_table("conc")

    ap("## 五、软件验收")
    judge_table("sw")

    ap("## 六、人工确认与文档归档")
    for key, label in REQUIRED_MANUAL:
        ok = man.get(key)
        ap(f"- {label}: {'✅' if ok else '❌ 未确认'}")
        if not ok:
            problems.append(f"人工确认未完成: {label}")
    ap(f"- 出厂测试报告编号: {man.get('factory_report_no') or '—'}")
    ap(f"- 备注: {man.get('notes') or '—'}")
    ap("")

    ap("## 验收结论")
    if problems:
        ap("**存在待解决问题：**")
        for p in problems:
            ap(f"- ❌ {p}")
    else:
        ap("**全部检查项通过 ✅（含人工确认项）**")
    ap("")
    ap("| 验收方代表 | 供货方代表 | 日期 |")
    ap("|---|---|---|")
    ap(f"| {man.get('acceptor') or ''} |  | {datetime.now():%Y-%m-%d} |")

    md = "\n".join(L)
    fname = f"验收报告_{datetime.now():%Y%m%d_%H%M%S}.md"
    (DATA / fname).write_text(md, encoding="utf-8")
    return fname, md


# --------------------------------------------------------------------------
# HTTP 服务
# --------------------------------------------------------------------------
class Handler(BaseHTTPRequestHandler):
    def log_message(self, *a):  # 安静模式
        pass

    def _send(self, code, body, ctype="application/json; charset=utf-8", extra=None):
        data = body if isinstance(body, bytes) else json.dumps(body, ensure_ascii=False).encode()
        self.send_response(code)
        self.send_header("Content-Type", ctype)
        self.send_header("Content-Length", str(len(data)))
        for k, v in (extra or {}).items():
            self.send_header(k, v)
        self.end_headers()
        self.wfile.write(data)

    def _body(self):
        try:
            ln = int(self.headers.get("Content-Length", 0))
            return json.loads(self.rfile.read(ln) or b"{}")
        except ValueError:
            return {}

    def do_GET(self):
        u = urlparse(self.path)
        if u.path in ("/", "/index.html"):
            try:
                html = (WEB / "index.html").read_bytes()
                self._send(200, html, "text/html; charset=utf-8")
            except OSError:
                self._send(500, {"error": "web/index.html 缺失"})
            return
        if u.path == "/api/state":
            self._send(200, {"cats": CATS, "checks": CHECKS, "results": load_results(),
                             "spec": load_spec(), "conn": load_conn()})
            return
        if u.path.startswith("/api/task/"):
            tid = u.path.rsplit("/", 1)[1]
            with TLOCK:
                t = TASKS.get(tid)
            if not t:
                self._send(404, {"error": "task not found"})
                return
            with TLOCK:
                snap = {"id": t["id"], "check": t["check"], "status": t["status"],
                        "metrics": dict(t["metrics"]),
                        "output": "\n".join(t["output"][-500:])}
            self._send(200, snap)
            return
        if u.path == "/api/report":
            try:
                fname, md = build_report()
                self._send(200, {"file": fname, "markdown": md})
            except Exception as e:  # noqa: BLE001
                self._send(500, {"error": f"{type(e).__name__}: {e}"})
            return
        if u.path == "/download":
            rel = parse_qs(u.query).get("p", [""])[0]
            f = (DATA / rel).resolve()
            if not str(f).startswith(str(DATA.resolve())) or not f.is_file():
                self._send(403, {"error": "forbidden"})
                return
            self._send(200, f.read_bytes(), "application/octet-stream",
                       {"Content-Disposition": f"attachment; filename*=UTF-8''{quote(f.name)}"})
            return
        self._send(404, {"error": "not found"})

    def do_POST(self):
        u = urlparse(self.path)
        body = self._body()
        if u.path == "/api/conn":
            conn = load_conn()
            conn.update({k: body[k] for k in conn if k in body})
            if str(conn.get("port", "")).isdigit() is False:
                self._send(400, {"error": "端口必须为数字"})
                return
            conn["port"] = int(conn["port"])
            save_json(CONN_FILE, conn)
            self._send(200, {"ok": True, "conn": conn})
            return
        if u.path == "/api/spec":
            spec = load_spec()
            for k in spec:
                if k in body:
                    try:
                        spec[k] = type(spec[k])(body[k])
                    except (TypeError, ValueError):
                        pass
            save_json(SPEC_FILE, spec)
            results = load_results()
            for rid, rec in results.items():
                if isinstance(rec, dict) and "metrics" in rec:
                    rec["judge"] = judge(rid, rec["metrics"], spec)
            save_results(results)
            self._send(200, {"ok": True, "spec": spec})
            return
        if u.path == "/api/test":
            conn = load_conn()
            try:
                r = subprocess.run(ssh_argv(conn, ["echo ACC_OK && hostname && nvidia-smi -L | head -1"]),
                                   capture_output=True, text=True, encoding="utf-8",
                                   errors="replace", timeout=25)
                self._send(200, {"ok": r.returncode == 0, "rc": r.returncode,
                                 "out": (r.stdout + r.stderr).strip()[:2000]})
            except subprocess.TimeoutExpired:
                self._send(200, {"ok": False, "out": "SSH 连接超时（25s）"})
            except Exception as e:  # noqa: BLE001
                self._send(200, {"ok": False, "out": f"{type(e).__name__}: {e}"})
            return
        if u.path == "/api/run":
            check = body.get("check")
            if check not in RUNNABLE:
                self._send(400, {"error": f"未知检查项 {check}"})
                return
            task = dispatch(check, body.get("params") or {})
            self._send(200, {"task": task["id"], "check": check})
            return
        if u.path == "/api/stop":
            tid = body.get("task", "")
            with TLOCK:
                t = TASKS.get(tid)
            if not t:
                self._send(404, {"error": "task not found"})
                return
            proc = t.get("_proc")
            if proc and proc.poll() is None:
                proc.terminate()
            t["status"] = "stopped"
            self._send(200, {"ok": True})
            return
        if u.path == "/api/manual":
            results = load_results()
            man = results.setdefault("manual", {})
            man.update({k: body[k] for k in body if k in {
                "appearance_ok", "contract_no", "course_image_ok", "accounts_ok",
                "factory_report_no", "sn_registered", "warranty_archived",
                "acceptor", "notes"}})
            save_results(results)
            self._send(200, {"ok": True, "manual": man})
            return
        self._send(404, {"error": "not found"})


def main():
    try:
        sys.stdout.reconfigure(encoding="utf-8")
    except Exception:  # noqa: BLE001
        pass
    srv = ThreadingHTTPServer((BIND, HTTP_PORT), Handler)
    url = f"http://{BIND}:{HTTP_PORT}"
    print(f"GPU 服务器验收控制台已启动: {url}  （仅监听本机回环，Ctrl+C 退出）")
    print(f"SSH: {SSH_BIN} | 数据目录: {DATA}")
    if os.environ.get("GPU_ACC_NO_BROWSER") != "1":
        threading.Timer(0.8, lambda: webbrowser.open(url)).start()
    try:
        srv.serve_forever()
    except KeyboardInterrupt:
        print("\n已退出。")


if __name__ == "__main__":
    main()
