#!/usr/bin/env bash
# 烤机/性能测试前置依赖检查（远程执行，只读）
set -u
echo "== 关键命令检查 =="
for c in nvidia-smi tmux git nvcc gcc make python3; do
  if command -v "$c" >/dev/null 2>&1; then echo "@@CMD_${c}=OK"; else echo "@@CMD_${c}=MISSING"; fi
done
echo "== GPU 状态 =="
nvidia-smi --query-gpu=index,name,temperature.gpu,power.draw,memory.total,driver_version --format=csv,noheader \
  || echo "@@ERROR=nvidia_smi_failed"
echo "== 根分区空间 =="
df -h / | tail -1
echo "== CUDA 工具链 =="
if command -v nvcc >/dev/null 2>&1; then
  nvcc --version | tail -2
else
  echo "(nvcc 未安装：gpu-burn / P2P 测试构建需要)"
fi
echo "== GPU 空闲利用率（应接近 0%）=="
nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader | tr '\n' ' '; echo
echo "== NVIDIA 内核模块 =="
lsmod | grep -i nvidia | head -5 || echo "(lsmod 无 nvidia 记录)"
