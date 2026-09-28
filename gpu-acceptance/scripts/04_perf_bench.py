#!/usr/bin/env python3
# -*- coding: utf-8 -*-
# ④ 性能实测（远程执行）：显存带宽（D2D 拷贝）+ FP32/TF32/FP16/FP8 matmul 算力
# 由本地控制台通过 `ssh <remote_python> -` 推送执行；remote_python 需为装有 PyTorch 的解释器
import sys
import time

try:
    import torch
except Exception as e:  # noqa: BLE001
    print("@@ERROR=import_torch_failed:" + str(e).replace("\n", " "))
    sys.exit(1)

print(f"@@TORCH={torch.__version__}")
print(f"@@GPU_COUNT={torch.cuda.device_count()}")
prop = torch.cuda.get_device_properties(0)
print(f"@@GPU0_NAME={prop.name}")
print(f"@@GPU0_MEM_GB={prop.total_memory / 1024**3:.1f}")
print(f"@@GPU0_CC={prop.major}.{prop.minor}")

dev = torch.device("cuda:0")
torch.cuda.synchronize()


def timed(op, iters, warmup=3):
    for _ in range(warmup):
        op()
    torch.cuda.synchronize()
    t0 = time.perf_counter()
    for _ in range(iters):
        op()
    torch.cuda.synchronize()
    return (time.perf_counter() - t0) / iters


# ---- 显存带宽：设备到设备大块拷贝（计入读+写双向流量）----
try:
    gib = 8 if prop.total_memory >= 24 * 1024**3 else 2
    n = gib * 1024**3 // 2  # fp16 元素数
    a = torch.empty(n, dtype=torch.float16, device=dev)
    b = torch.empty_like(a)
    t = timed(lambda: b.copy_(a), 20)
    bw = 2 * n * 2 / t / 1e9
    print(f"@@MEM_BW_GBPS={bw:.1f}")
    print(f"显存带宽（D2D 拷贝，读+写计）: {bw:.1f} GB/s，块大小 {gib} GiB")
    del a, b
    torch.cuda.empty_cache()
except Exception as e:  # noqa: BLE001
    print("@@MEM_BW_FAILED=" + type(e).__name__)


# ---- 算力：方阵 matmul（2*n^3 浮点操作）----
def bench_mm(dtype, n, iters, label, metric):
    try:
        a = torch.randn(n, n, dtype=torch.float32, device=dev)
        if dtype is not torch.float32:
            a = a.to(dtype)
        b = (-a).contiguous()
        t = timed(lambda: a @ b, iters)
        tf = 2 * n**3 / t / 1e12
        print(f"@@{metric}={tf:.1f}")
        print(f"{label}: {tf:.1f} TFLOPS（{n}x{n}，{iters} 次均值）")
        del a, b
        torch.cuda.empty_cache()
    except Exception as e:  # noqa: BLE001
        print(f"@@{metric}_FAILED={type(e).__name__}")


torch.backends.cuda.matmul.allow_tf32 = False
bench_mm(torch.float32, 8192, 15, "FP32 算力", "FP32_TFLOPS")
torch.backends.cuda.matmul.allow_tf32 = True
bench_mm(torch.float32, 8192, 15, "TF32 算力(Tensor Core)", "TF32_TFLOPS")
bench_mm(torch.float16, 8192, 15, "FP16 算力(Tensor Core)", "FP16_TFLOPS")

# ---- FP8（Ada 支持，torch._scaled_mm）----
try:
    if not hasattr(torch, "float8_e4m3fn") or not hasattr(torch, "_scaled_mm"):
        raise RuntimeError("torch 版本不支持 _scaled_mm/float8_e4m3fn")
    n = 8192
    fa = torch.randn(n, n, device=dev).clamp(-2, 2).to(torch.float8_e4m3fn)
    fb = torch.randn(n, n, device=dev).clamp(-2, 2).t().contiguous().t().to(torch.float8_e4m3fn)
    sa = torch.ones(1, device=dev)
    sb = torch.ones(1, device=dev)

    def op():
        return torch._scaled_mm(fa, fb, scale_a=sa, scale_b=sb, out_dtype=torch.float16)

    try:
        op()
    except TypeError:  # 旧签名：位置参数
        def op():  # noqa: F811
            return torch._scaled_mm(fa, fb, sa, sb, out_dtype=torch.float16)

        op()
    t = timed(op, 15)
    tf = 2 * n**3 / t / 1e12
    print(f"@@FP8_TFLOPS={tf:.1f}")
    print(f"FP8 算力(_scaled_mm e4m3): {tf:.1f} TFLOPS")
except Exception as e:  # noqa: BLE001
    print(f"@@FP8_TFLOPS_FAILED={type(e).__name__}")
    print("（FP8 实测失败不影响其他项；可用 gpu-burn 或 TensorRT 补充验证）")

print("@@PERF_DONE=1")
