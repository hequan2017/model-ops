#!/usr/bin/env bash
# ③ 停止烤机：结束负载与监控会话，保留全部日志（远程执行）
set -u
DIR=/root/acceptance
tmux kill-session -t acc_burn 2>/dev/null && echo "负载会话 acc_burn 已结束" || echo "负载会话不存在"
tmux kill-session -t acc_mon 2>/dev/null && echo "监控会话 acc_mon 已结束" || echo "监控会话不存在"
nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader | tr '\n' ' '; echo "（停机后利用率）"
if [ -f "$DIR/burnin_start_epoch" ]; then
  start=$(cat "$DIR/burnin_start_epoch"); now=$(date +%s)
  echo "@@TOTAL_MIN=$(( (now - start) / 60 ))"
fi
echo "@@STOPPED=yes"
echo "日志保留在 $DIR：burnin.log / burnin_monitor.csv"
