#!/usr/bin/env python3
# -*- coding: utf-8 -*-
# ④ 并发压测（本地执行）：模拟师生并发访问推理/教学服务，统计响应时间与错误率
# 用法: python 04_concurrency_test.py <url> <时长秒> <并发数> [POST的JSON体] [--allow-private]
# 目标仅允许 http/https；解析后阻断链路本地/云元数据地址段；私网/环回目标需显式 --allow-private；不跟随重定向
# 示例: python 04_concurrency_test.py http://<服务器IP>:8000/v1/completions 120 10 \
#         '{"model":"模型名","prompt":"你好","max_tokens":16}'
import ipaddress
import json
import socket
import statistics
import sys
import threading
import time
import urllib.error
import urllib.request

BLOCKED_NETS = [ipaddress.ip_network("169.254.0.0/16"),     # 链路本地 / 云元数据
                ipaddress.ip_network("0.0.0.0/8"),
                ipaddress.ip_network("fe80::/10")]
PRIVATE_NETS = [ipaddress.ip_network("10.0.0.0/8"),
                ipaddress.ip_network("172.16.0.0/12"),
                ipaddress.ip_network("192.168.0.0/16"),
                ipaddress.ip_network("127.0.0.0/8"),
                ipaddress.ip_network("::1/128"),
                ipaddress.ip_network("fc00::/7")]


def check_target(url, allow_private):
    from urllib.parse import urlparse
    p = urlparse(url)
    if p.scheme not in ("http", "https") or not p.hostname or p.username:
        sys.exit(f"目标 URL 不合法（仅允许 http/https 且不含 userinfo）: {url}")
    try:
        infos = socket.getaddrinfo(p.hostname, p.port or (443 if p.scheme == "https" else 80),
                                   proto=socket.IPPROTO_TCP)
    except socket.gaierror as e:
        sys.exit(f"目标主机解析失败: {e}")
    for info in infos:
        ip = ipaddress.ip_address(info[4][0])
        if any(ip in n for n in BLOCKED_NETS):
            sys.exit(f"目标地址 {ip} 位于禁止访问的链路本地/元数据段")
        if any(ip in n for n in PRIVATE_NETS) and not allow_private:
            sys.exit(f"目标地址 {ip} 为私网/环回地址，压测内网服务请追加 --allow-private")


class NoRedirect(urllib.request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        return None  # 拒绝重定向，防止绕过目标校验


def main():
    args = [a for a in sys.argv[1:] if a != "--allow-private"]
    allow_private = "--allow-private" in sys.argv
    if len(args) < 3:
        print("用法: 04_concurrency_test.py <url> <duration_s> <users> [json_body] [--allow-private]")
        sys.exit(2)
    url, duration, conc = args[0], int(args[1]), int(args[2])
    method, body = "GET", None
    if len(args) > 3 and args[3].strip():
        method = "POST"
        try:
            body = json.loads(args[3])
        except ValueError:
            body = args[3]
    check_target(url, allow_private)

    opener = urllib.request.build_opener(NoRedirect)
    lat, errors = [], 0
    lock = threading.Lock()
    stop_at = time.time() + duration
    print(f"目标: {method} {url} | 并发 {conc} | 时长 {duration}s | 开始 {time.strftime('%H:%M:%S')}")

    def worker():
        nonlocal errors
        while time.time() < stop_at:
            t0 = time.perf_counter()
            try:
                data = None
                if body is not None:
                    data = json.dumps(body).encode() if isinstance(body, (dict, list)) else str(body).encode()
                req = urllib.request.Request(
                    url, data=data, method=method,
                    headers={"Content-Type": "application/json"})
                with opener.open(req, timeout=180) as r:
                    r.read()
                with lock:
                    lat.append((time.perf_counter() - t0) * 1000)
            except urllib.error.HTTPError:
                with lock:  # 服务端返回了状态码，计入错误但保留时延
                    lat.append((time.perf_counter() - t0) * 1000)
                    errors += 1
            except Exception:  # noqa: BLE001
                with lock:
                    lat.append((time.perf_counter() - t0) * 1000)
                    errors += 1

    ths = [threading.Thread(target=worker, daemon=True) for _ in range(conc)]
    for t in ths:
        t.start()
    for t in ths:
        t.join()

    lat.sort()

    def pct(p):
        return lat[min(len(lat) - 1, int(len(lat) * p))] if lat else -1

    print(f"@@REQS={len(lat)}")
    print(f"@@ERRORS={errors}")
    print(f"@@RPS={len(lat) / duration:.2f}")
    print(f"@@P50_MS={pct(0.50):.1f}")
    print(f"@@P95_MS={pct(0.95):.1f}")
    print(f"@@P99_MS={pct(0.99):.1f}")
    print(f"@@AVG_MS={statistics.mean(lat):.1f}" if lat else "@@AVG_MS=-1")
    print(f"结果: 请求 {len(lat)} | 错误 {errors} | RPS {len(lat)/duration:.2f} | "
          f"P50 {pct(0.50):.0f}ms | P95 {pct(0.95):.0f}ms | P99 {pct(0.99):.0f}ms")


if __name__ == "__main__":
    main()
