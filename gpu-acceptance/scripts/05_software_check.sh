#!/usr/bin/env bash
# ⑤ 软件验收（远程执行）：驱动/CUDA/nvcc/Docker 镜像 + PyTorch/TensorFlow GPU 验证
# 通过 env ACC_PY=<解释器> 指定待验证的 Python（在连接设置中的 remote_python 配置）
set -u
PY=${ACC_PY:-python3}
echo "== 驱动 / CUDA 运行时 =="
nvidia-smi | head -4
drv=$(nvidia-smi --query-gpu=driver_version --format=csv,noheader 2>/dev/null | head -1)
cuda=$(nvidia-smi 2>/dev/null | grep -o 'CUDA Version: [0-9.]*' | head -1 | awk '{print $3}')
echo "@@DRIVER=$drv"
echo "@@CUDA=$cuda"
if command -v nvcc >/dev/null 2>&1; then
  echo "@@NVCC=$(nvcc --version | grep -o 'release [0-9.]*' | awk '{print $2}')"
else
  echo "@@NVCC=未安装"
fi

echo "== Docker 与课程镜像 =="
if command -v docker >/dev/null 2>&1; then
  echo "@@DOCKER=$(docker --version 2>/dev/null | head -1)"
  docker images --format '{{.Repository}}:{{.Tag}} ({{.Size}})' 2>/dev/null | head -20
else
  echo "@@DOCKER=未安装"
fi

echo "== Python 环境 =="
command -v conda >/dev/null 2>&1 && conda env list 2>/dev/null
echo "待验证解释器: $PY"

"$PY" - <<'EOF'
import sys
print(f"@@PY_VER={sys.version.split()[0]}")
try:
    import torch
    x = torch.randn(1024, 1024, device="cuda")
    _ = (x @ x).sum().item()   # 真实 GPU 计算验证
    print(f"@@TORCH={torch.__version__}")
    print(f"@@TORCH_CUDA_OK={str(torch.cuda.is_available()).lower()}")
    print(f"@@TORCH_GPU_N={torch.cuda.device_count()}")
    print("PyTorch GPU 矩阵运算验证: 通过")
except Exception as e:
    print(f"@@TORCH=FAILED:{type(e).__name__}:{e}"[:200])
try:
    import tensorflow as tf
    gpus = tf.config.list_physical_devices("GPU")
    print(f"@@TF={tf.__version__}")
    print(f"@@TF_GPU_N={len(gpus)}")
except Exception as e:
    print(f"@@TF=FAILED:{type(e).__name__}"[:200])
EOF
echo "== NVIDIA 持久化 / DCGM-exporter（可选监控项）=="
(nvidia-smi -q 2>/dev/null | grep -i 'Persistence Mode' | head -1) || true
systemctl is-active dcgm-exporter 2>/dev/null || echo "dcgm-exporter: 未部署（可选，Prometheus 监控用）"
