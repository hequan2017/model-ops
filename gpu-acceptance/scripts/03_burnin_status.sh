#!/usr/bin/env bash
# ③ 烤机状态：进度 / 温度功耗峰值 / ECC 计数 / Xid 日志（远程执行，只读）
set -u
DIR=/root/acceptance
if [ ! -f "$DIR/burnin_start_epoch" ]; then
  echo "@@RUNNING=never"
  echo "尚未启动烤机（找不到 $DIR/burnin_start_epoch）"
  exit 0
fi
start=$(cat "$DIR/burnin_start_epoch")
dur=$(cat "$DIR/burnin_duration_min" 2>/dev/null || echo 4320)
now=$(date +%s)
el=$(( (now - start) / 60 ))
awk -v e="$el" -v d="$dur" 'BEGIN{p=e*100/d; if(p>100)p=100; printf "@@ELAPSED_MIN=%d\n@@DURATION_MIN=%d\n@@PROGRESS=%.1f\n", e, d, p}'
if tmux has-session -t acc_burn 2>/dev/null; then echo "@@RUNNING=yes"; else echo "@@RUNNING=no"; fi

echo "== 温度 / 功耗峰值 与 ECC 计数（来自监控采样）=="
if [ -s "$DIR/burnin_monitor.csv" ]; then
  awk -F', ' 'NR>1{
      w=$4; gsub(/ ?W/,"",w); uu=$6; gsub(/ ?%/,"",uu);
      idx=$2+0;
      if($3+0>t[idx])t[idx]=$3+0;
      if(w+0>p[idx])p[idx]=w+0;
      c[idx]=$8+0; n[idx]=$9+0; cnt++
    }
    END{
      tc=0; tu=0
      for(i in t){ printf "@@GPU%d_MAX_TEMP=%d\n@@GPU%d_MAX_POWER=%d\n", i, t[i], i, p[i]; tc+=c[i]; tu+=n[i] }
      printf "@@ECC_CORR_TOTAL=%d\n@@ECC_UNCORR_TOTAL=%d\n@@MON_SAMPLES=%d\n", tc, tu, cnt
    }' "$DIR/burnin_monitor.csv"
fi

echo "== gpu-burn 输出尾部 =="
tail -n 6 "$DIR/burnin.log" 2>/dev/null
echo "== 内核日志 Xid/NVRM（应为空）=="
dmesg -T 2>/dev/null | grep -iE 'xid|nvrm.*error' | tail -n 10 || echo "(无 Xid 记录)"
