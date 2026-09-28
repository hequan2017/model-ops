#!/usr/bin/env bash
# ③ 72h 满载烤机启动（远程执行）
# 用法: burnin_start.sh [时长分钟，默认 4320 = 72h]
# 负载与监控均运行在 tmux 会话中，断开 SSH 不中断；日志保存在 /root/acceptance/
set -u
DUR=${1:-4320}
DIR=/root/acceptance
mkdir -p "$DIR"

# --- 定位或构建 gpu-burn ---
GB=""
for c in /root/gpu-burn/gpu_burn /usr/local/bin/gpu_burn; do
  [ -x "$c" ] && GB="$c" && break
done
if [ -z "$GB" ] && command -v gpu_burn >/dev/null 2>&1; then GB=$(command -v gpu_burn); fi
if [ -z "$GB" ]; then
  echo "未找到 gpu-burn，尝试克隆并编译（需要 git + nvcc + make）..."
  if command -v git >/dev/null 2>&1 && command -v nvcc >/dev/null 2>&1 && command -v make >/dev/null 2>&1; then
    cd /root && rm -rf gpu-burn
    if ! git clone --depth 1 https://github.com/wilicc/gpu-burn.git 2>&1 | tail -2; then
      echo "@@ERROR=clone_failed（GitHub 访问受限时可手动下载 gpu-burn 放到 /root/gpu-burn 后重试）"
      exit 1
    fi
    if ! make -C /root/gpu-burn 2>&1 | tail -5; then
      echo "@@ERROR=build_failed"; exit 1
    fi
    GB=/root/gpu-burn/gpu_burn
  else
    echo "@@ERROR=missing_git_or_nvcc"; exit 1
  fi
fi
echo "使用 gpu-burn: $GB"

command -v tmux >/dev/null 2>&1 || { echo "@@ERROR=no_tmux"; exit 1; }

# --- 清理旧会话并启动 ---
tmux kill-session -t acc_burn 2>/dev/null
tmux kill-session -t acc_mon 2>/dev/null
date +%s > "$DIR/burnin_start_epoch"
echo "$DUR" > "$DIR/burnin_duration_min"
rm -f "$DIR/burnin.log" "$DIR/burnin_monitor.csv"

GBDIR=$(dirname "$GB")
tmux new-session -d -s acc_burn "cd '$GBDIR' && ./gpu_burn $DUR > $DIR/burnin.log 2>&1"
tmux new-session -d -s acc_mon "while true; do nvidia-smi --query-gpu=timestamp,index,temperature.gpu,power.draw,clocks.sm,utilization.gpu,memory.used,ecc.errors.corrected.volatile.total,ecc.errors.uncorrected.volatile.total --format=csv,noheader >> $DIR/burnin_monitor.csv 2>&1; sleep 30; done"
sleep 8

if tmux has-session -t acc_burn 2>/dev/null; then
  echo "@@STARTED=yes"
else
  echo "@@STARTED=no"
  echo "-- burnin.log --"; cat "$DIR/burnin.log" 2>/dev/null
  exit 1
fi
echo "@@DURATION_MIN=$DUR"
echo "@@GPU_BURN=$GB"
echo "@@LOG_DIR=$DIR"
echo "烤机已在后台启动：负载会话 acc_burn，监控会话 acc_mon（30 秒采样一次）"
echo "查看: tmux attach -t acc_burn | 日志: tail -f $DIR/burnin.log"
