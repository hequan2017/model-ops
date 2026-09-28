#!/usr/bin/env bash
# ③ ECC 日志核查：ECC 模式 / 易失与累计错误计数 / 内核日志（远程执行，只读）
# 验收要求：ECC 事件 0 错误
set -u
echo "== ECC 模式与错误计数（每 GPU）=="
nvidia-smi --query-gpu=index,ecc.mode.current,ecc.errors.corrected.volatile.total,ecc.errors.uncorrected.volatile.total,ecc.errors.corrected.aggregate.total,ecc.errors.uncorrected.aggregate.total --format=csv,noheader 2>/dev/null | \
while IFS=',' read -r i mode cv uv ca ua; do
  echo "@@ECC|${i}|${mode}|${cv}|${uv}|${ca}|${ua}"
done
nvidia-smi --query-gpu=ecc.errors.corrected.aggregate.total,ecc.errors.uncorrected.aggregate.total --format=csv,noheader,nounits 2>/dev/null | \
awk -F', ' '{tc+=$1; tu+=$2} END{printf "@@ECC_CORR_TOTAL=%d\n@@ECC_UNCORR_TOTAL=%d\n", tc, tu}'
echo "== 内核日志 ECC / Xid（应为空）=="
dmesg -T 2>/dev/null | grep -iE 'xid|ecc' | tail -n 30 || echo "(无相关内核日志)"
echo "== DCGM（如已安装，可运行深度诊断 dcgmi diag -r 3）=="
command -v dcgmi >/dev/null 2>&1 && echo "dcgmi 已安装：可执行 dcgmi diag -r 3" || echo "dcgmi 未安装（可选，用于 DCGM 诊断与 dcgm-exporter 监控）"
