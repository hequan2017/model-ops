#!/usr/bin/env bash
# ① 开箱核对：GPU 型号/序列号、整机/主板/BIOS、网卡、NVMe 清单（远程执行，只读）
set -u
echo "== 主机 =="
hostname; date
echo "== GPU 清单（型号 / 序列号 / UUID / PCI / 显存 / 驱动）=="
if ! nvidia-smi --query-gpu=index,name,serial,uuid,pci.bus_id,memory.total,driver_version --format=csv,noheader 2>/dev/null; then
  echo "@@ERROR=nvidia_smi_failed"; exit 1
fi
nvidia-smi --query-gpu=index,name,serial,uuid,pci.bus_id,memory.total,driver_version --format=csv,noheader 2>/dev/null | \
while IFS=',' read -r idx name serial uuid bus mem drv; do
  echo "@@GPU|${idx}|${name}|${serial}|${uuid}|${bus}|${mem}|${drv}"
done
echo "== 整机信息 =="
echo "厂商: $(dmidecode -s system-manufacturer 2>/dev/null)"
echo "型号: $(dmidecode -s system-product-name 2>/dev/null)"
echo "@@SYSTEM_SN=$(dmidecode -s system-serial-number 2>/dev/null)"
echo "== 主板 / BIOS =="
dmidecode -t 2 2>/dev/null | grep -E 'Manufacturer|Product Name|Serial Number'
dmidecode -t 0 2>/dev/null | grep -E 'Vendor|Version|Release'
echo "== PCIe 设备（GPU / 网卡）=="
lspci -nn 2>/dev/null | grep -Ei 'nvidia|ethernet|infiniband|mellanox' | head -30
echo "== NVMe / 磁盘 =="
if command -v nvme >/dev/null 2>&1; then
  nvme list 2>/dev/null || lsblk -d -o NAME,SIZE,MODEL,SERIAL
else
  lsblk -d -o NAME,SIZE,MODEL 2>/dev/null
fi
echo "== GPU 物理拓扑 =="
nvidia-smi topo -m 2>/dev/null | head -12
