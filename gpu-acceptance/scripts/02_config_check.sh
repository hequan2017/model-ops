#!/usr/bin/env bash
# ② 配置核对：CPU / 内存 / 磁盘 / GPU 数量与型号采集（远程执行，只读）
# 输出 @@KEY=VALUE 供本地控制台与验收标准自动比对
set -u
echo "== CPU =="
model=$(lscpu | awk -F: '/Model name/{gsub(/^ +/,"",$2); print $2; exit}')
sockets=$(lscpu | awk -F: '/^Socket\(s\)/{gsub(/^ +/,"",$2); print $2; exit}')
cps=$(lscpu | awk -F: '/Core\(s\) per socket/{gsub(/^ +/,"",$2); print $2; exit}')
tpc=$(lscpu | awk -F: '/Thread\(s\) per core/{gsub(/^ +/,"",$2); print $2; exit}')
logical=$(nproc)
echo "型号: $model ( $($sockets) 路 × $($cps) 核 × $($tpc) 线程 )"
echo "@@CPU_MODEL=$model"
echo "@@SOCKETS=$sockets"
echo "@@CORES_PER_SOCKET=$cps"
echo "@@THREADS_PER_CORE=$tpc"
echo "@@LOGICAL_CPUS=$logical"

echo "== 内存 =="
mem_kb=$(free -k | awk '/^Mem:/{print $2}')
mem_gb=$(awk -v k="$mem_kb" 'BEGIN{printf "%.1f", k/1024/1024}')
dimms=$(dmidecode -t memory 2>/dev/null | grep -E '^[[:space:]]+Size:[[:space:]]+[0-9]+ GB' | awk '{s+=$2} END{print s+0}')
free -h | head -2
echo "@@MEM_TOTAL_GB=$mem_gb"
echo "@@MEM_DIMMS_GB=$dimms"

echo "== 磁盘 =="
lsblk -ndo NAME,SIZE,TYPE,MOUNTPOINT
echo "-- 文件系统 --"
df -h | grep -Ev 'tmpfs|overlay|loop|squashfs'
for m in / /data /data2 /data3; do
  sz=$(df -BG --output=size "$m" 2>/dev/null | tail -1 | tr -dc '0-9')
  if [ -n "$sz" ]; then echo "@@DISK_${m//\//_}_GB=$sz"; fi
done

echo "== GPU =="
gcount=$(nvidia-smi --list-gpus 2>/dev/null | wc -l)
gmodel=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1)
gmem=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader 2>/dev/null | head -1)
echo "@@GPU_COUNT=$gcount"
echo "@@GPU_MODEL=$gmodel"
echo "@@GPU_MEM=$gmem"
