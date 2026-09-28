#!/usr/bin/env bash
# ④ 性能实测（强制项）：GPU P2P 带宽（远程执行）
# 优先使用现成 p2pBandwidthLatencyTest；否则从 cuda-samples 稀疏克隆并编译（需要 git + nvcc + make）
set -u
echo "== GPU 拓扑 =="
nvidia-smi topo -m 2>/dev/null

EXE=""
for c in /root/cuda-samples/bin/x86_64/linux/release/p2pBandwidthLatencyTest /root/p2pBandwidthLatencyTest /usr/local/bin/p2pBandwidthLatencyTest; do
  [ -x "$c" ] && EXE="$c" && break
done
[ -z "$EXE" ] && command -v p2pBandwidthLatencyTest >/dev/null 2>&1 && EXE=$(command -v p2pBandwidthLatencyTest)

if [ -z "$EXE" ]; then
  echo "未找到 p2pBandwidthLatencyTest，尝试从 cuda-samples 构建（稀疏克隆，仅拉取所需目录）..."
  if command -v git >/dev/null 2>&1 && command -v nvcc >/dev/null 2>&1 && command -v make >/dev/null 2>&1; then
    cd /root && rm -rf cuda-samples
    if ! git clone --depth 1 --filter=blob:none --sparse https://github.com/NVIDIA/cuda-samples.git 2>&1 | tail -2; then
      echo "@@ERROR=clone_failed（GitHub 访问受限时，可手工拷贝 cuda-samples 到 /root/cuda-samples 后重试）"
      nvidia-smi topo -m; exit 1
    fi
    cd cuda-samples || { echo "@@ERROR=clone_failed"; exit 1; }
    git sparse-checkout set Common Samples/5_Domain_Specific/p2pBandwidthLatencyTest 2>/dev/null \
      || git sparse-checkout set Samples/5_Domain_Specific/p2pBandwidthLatencyTest 2>/dev/null || true
    B=$(find . -type d -name p2pBandwidthLatencyTest | head -1)
    if [ -z "$B" ]; then echo "@@ERROR=sample_dir_not_found"; exit 1; fi
    echo "构建目录: $B"
    make -C "$B" 2>&1 | tail -3
    for c in "$B/../../bin/x86_64/linux/release/p2pBandwidthLatencyTest" "$B/p2pBandwidthLatencyTest"; do
      [ -x "$c" ] && EXE="$c" && break
    done
  else
    echo "@@ERROR=missing_git_or_nvcc"
    echo "拓扑信息已输出，可先人工核对；构建工具齐全后重试可获得 P2P 带宽矩阵。"
    exit 1
  fi
fi

if [ -z "$EXE" ]; then echo "@@ERROR=build_failed"; exit 1; fi
echo "使用: $EXE"
OUT=$("$EXE" 2>&1)
echo "$OUT"
echo "$OUT" | awk '
  /Unidirectional|Bidirectional/ {f=1}
  f && /[0-9]+\.[0-9]+/ {
    for (i = 1; i <= NF; i++) {
      if ($i ~ /^[0-9]+\.?[0-9]*$/ && $i + 0 > 5 && $i + 0 > m) m = $i + 0
    }
  }
  END { if (m > 0) printf "@@P2P_MAX_GBPS=%.2f\n", m }'
echo "说明：无 NVLink 的机型（如 L40）P2P 走 PCIe（Gen4 x16 理论单向 32GB/s，实测一般 20~26GB/s）；"
echo "      输出矩阵中 P2P=Disabled 的 GPU 对为拓扑/固件限制，以人工判读为准。"
