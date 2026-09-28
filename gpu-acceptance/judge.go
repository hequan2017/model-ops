package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func num(v interface{}) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case int:
		return float64(x), true
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(x), 64)
		return f, err == nil
	}
	return 0, false
}

func str(v interface{}) string {
	switch x := v.(type) {
	case string:
		return x
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", x)
	}
}

func boolp(b bool) *bool { return &b }

// judge 指标 vs 验收标准
func judge(checkID string, m map[string]interface{}, spec Spec) []JudgeRow {
	rows := []JudgeRow{}
	add := func(label string, actual interface{}, expected string, ok *bool) {
		rows = append(rows, JudgeRow{Label: label, Actual: str(actual), Expected: expected, Pass: ok})
	}
	cmp := func(actual interface{}, expected string, cond bool, has bool) *bool {
		if !has {
			return nil
		}
		return boolp(cond)
	}

	switch checkID {
	case "cfg":
		v, ok := num(m["GPU_COUNT"])
		add("GPU 数量", m["GPU_COUNT"], strconv.Itoa(spec.GPUCount),
			cmp(nil, "", ok && v == float64(spec.GPUCount), ok))
		model := str(m["GPU_MODEL"])
		add("GPU 型号", m["GPU_MODEL"], "包含 "+spec.GPUModelContains,
			cmp(nil, "", model != "" && strings.Contains(strings.ToLower(model), strings.ToLower(spec.GPUModelContains)), model != ""))
		v, ok = num(m["LOGICAL_CPUS"])
		add("CPU 逻辑核数", m["LOGICAL_CPUS"], strconv.Itoa(spec.LogicalCores),
			cmp(nil, "", ok && v == float64(spec.LogicalCores), ok))
		mg, hasMg := num(m["MEM_TOTAL_GB"])
		tol := float64(spec.MemTolPct) / 100.0
		add("内存容量 GB (free 可用总量)", m["MEM_TOTAL_GB"],
			fmt.Sprintf("%d ±%d%%", spec.MemGB, spec.MemTolPct),
			cmp(nil, "", hasMg && abs(mg-float64(spec.MemGB)) <= float64(spec.MemGB)*tol, hasMg))
		key := "DISK_" + strings.ReplaceAll(spec.DataDiskMount, "/", "_") + "_GB"
		dv, hasDv := num(m[key])
		add(fmt.Sprintf("数据盘 %s 容量 GB", spec.DataDiskMount), m[key],
			fmt.Sprintf("≥%d", spec.DataDiskMinGB),
			cmp(nil, "", hasDv && dv >= float64(spec.DataDiskMinGB), hasDv))

	case "preflight":
		keys := sortedKeys(m)
		for _, k := range keys {
			if strings.HasPrefix(k, "CMD_") {
				v := str(m[k])
				add("命令 "+k[4:], v, "OK", boolp(v == "OK"))
			}
		}

	case "perf":
		mb, hasMb := num(m["MEM_BW_GBPS"])
		mbE := spec.MemBwGbps
		r := spec.MemBwMinRatio
		add("显存带宽 GB/s（D2D 拷贝，读+写）", m["MEM_BW_GBPS"],
			fmt.Sprintf("≥%.0f×%.2f=%.0f", mbE, r, mbE*r),
			cmp(nil, "", hasMb && mb >= mbE*r, hasMb))
		type pair struct {
			specKey float64
			mkey    string
			label   string
		}
		for _, p := range []pair{
			{spec.FP32TFlops, "FP32_TFLOPS", "FP32 算力 TFLOPS"},
			{spec.TF32TFlops, "TF32_TFLOPS", "TF32 算力 TFLOPS（参考）"},
			{spec.FP16TFlops, "FP16_TFLOPS", "FP16 算力 TFLOPS（参考）"},
			{spec.FP8TFlops, "FP8_TFLOPS", "FP8 算力 TFLOPS"},
		} {
			av, hasAv := num(m[p.mkey])
			rr := spec.FlopsMinRatio
			add(p.label, m[p.mkey], fmt.Sprintf("≥%.1f×%.2f=%.1f", p.specKey, rr, p.specKey*rr),
				cmp(nil, "", hasAv && av >= p.specKey*rr && !strings.Contains(str(m[p.mkey]), "FAILED"), hasAv))
		}

	case "p2p":
		v, has := num(m["P2P_MAX_GBPS"])
		add("P2P/总线带宽峰值 GB/s（自动提取，矩阵需人工复核）", m["P2P_MAX_GBPS"],
			fmt.Sprintf("≥%.0f", spec.P2PMinGbps),
			cmp(nil, "", has && v >= spec.P2PMinGbps, has))

	case "burn_status":
		if str(m["RUNNING"]) == "never" {
			add("状态", "未启动烤机", "-", nil)
		} else {
			add("负载运行中", m["RUNNING"], "yes", boolp(str(m["RUNNING"]) == "yes"))
			el, hasEl := num(m["ELAPSED_MIN"])
			du, hasDu := num(m["DURATION_MIN"])
			if hasEl && hasDu && du > 0 {
				add("已运行 / 计划（分钟）",
					fmt.Sprintf("%s / %s", str(m["ELAPSED_MIN"]), str(m["DURATION_MIN"])),
					fmt.Sprintf("≥%d", spec.BurnDurationMin),
					boolp(el >= float64(spec.BurnDurationMin)))
			}
			for _, k := range []string{"ECC_CORR_TOTAL", "ECC_UNCORR_TOTAL"} {
				v, has := num(m[k])
				add(k, m[k], "=0", cmp(nil, "", has && v == 0, has))
			}
			tmax := -1.0
			for k, v := range m {
				if strings.HasSuffix(k, "_MAX_TEMP") {
					if tv, ok := num(v); ok && tv > tmax {
						tmax = tv
					}
				}
			}
			if tmax >= 0 {
				add("GPU 峰值温度 ℃", fmt.Sprintf("%.0f", tmax),
					fmt.Sprintf("≤%d", spec.GPUTempMax),
					boolp(tmax <= float64(spec.GPUTempMax)))
			}
		}

	case "burn_start":
		add("烤机启动", m["STARTED"], "yes", boolp(str(m["STARTED"]) == "yes"))

	case "burn_stop":
		add("烤机停止", m["STOPPED"], "yes", boolp(str(m["STOPPED"]) == "yes"))

	case "ecc":
		c, hasC := num(m["ECC_CORR_TOTAL"])
		u, hasU := num(m["ECC_UNCORR_TOTAL"])
		add("ECC 可纠正错误（累计合计）", m["ECC_CORR_TOTAL"], "=0", cmp(nil, "", hasC && c == 0, hasC))
		add("ECC 不可纠正错误（累计合计）", m["ECC_UNCORR_TOTAL"], "=0", cmp(nil, "", hasU && u == 0, hasU))

	case "sw":
		add("NVIDIA 驱动", m["DRIVER"], "已安装", boolp(str(m["DRIVER"]) != ""))
		add("CUDA 运行时", m["CUDA"], "已安装", boolp(str(m["CUDA"]) != ""))
		add("PyTorch", m["TORCH"], "可导入且 CUDA 可用", boolp(str(m["TORCH_CUDA_OK"]) == "true"))
		tf := str(m["TF"])
		if tf == "" || strings.HasPrefix(tf, "FAILED") {
			add("TensorFlow", m["TF"], "可导入（课程需要时）", nil)
		} else {
			add("TensorFlow", m["TF"], "可导入（课程需要时）", boolp(true))
		}

	case "conc":
		v, has := num(m["P95_MS"])
		add("P95 响应 ms", m["P95_MS"], fmt.Sprintf("≤%.0f", spec.P95MaxMs),
			cmp(nil, "", has && v <= spec.P95MaxMs, has))
		er, hasEr := num(m["ERRORS"])
		add("错误请求数", m["ERRORS"], fmt.Sprintf("≤%d", spec.ErrMax),
			cmp(nil, "", hasEr && er <= float64(spec.ErrMax), hasEr))
	}
	return rows
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func sortedKeys(m map[string]interface{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// saveResult 保存任务结果到 results.json 与日志文件
func saveResult(t *Task) {
	rid := t.Check
	logRel := ""
	name := fmt.Sprintf("%s_%s.log", rid, t.ID)
	t.mu.Lock()
	out := strings.Join(t.Output, "\n")
	metrics := map[string]interface{}{}
	for k, v := range t.Metrics {
		metrics[k] = v
	}
	status := t.Status
	t.mu.Unlock()
	if err := os.WriteFile(filepath.Join(logsDir, name), []byte(out), 0o644); err == nil {
		logRel = "logs/" + name
	}
	rec := ResultRec{
		TS:      time.Now().Format("2006-01-02 15:04:05"),
		Status:  status,
		Metrics: metrics,
		Log:     logRel,
		Judge:   judge(rid, metrics, loadSpec()),
	}
	res := loadResults()
	res[rid] = rec
	saveResults(res)
}
