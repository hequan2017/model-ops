package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var requiredManual = []struct{ key, label string }{
	{"appearance_ok", "① 开箱外观无运输损伤"},
	{"course_image_ok", "⑤ 课程镜像已部署验证"},
	{"accounts_ok", "⑤ 师生账号已开通验证"},
	{"sn_registered", "⑥ GPU 序列号已注册"},
	{"warranty_archived", "⑥ 质保凭证已归档"},
}

func manualMap(res Results) map[string]interface{} {
	m, _ := res["manual"].(map[string]interface{})
	return m
}

func mark(p *bool) string {
	if p == nil {
		return "—"
	}
	if *p {
		return "✅"
	}
	return "❌"
}

func buildReport() (string, string, error) {
	conn, spec, res := loadConn(), loadSpec(), loadResults()
	man := manualMap(res)
	problems := []string{}
	var L []string
	ap := func(s string) { L = append(L, s) }

	ap("# GPU 服务器验收报告")
	ap("")
	ap(fmt.Sprintf("- 验收对象: `%s@%s:%d`", conn.User, conn.Host, conn.Port))
	ap(fmt.Sprintf("- 整机标称: %d × %s，%d 逻辑核，%dGB 内存",
		spec.GPUCount, spec.GPUModelContains, spec.LogicalCores, spec.MemGB))
	ap(fmt.Sprintf("- 报告生成: %s", time.Now().Format("2006-01-02 15:04:05")))
	ap("")

	judgeTable := func(rid string) {
		rec := getResultRec(res, rid)
		if rec == nil {
			ap(fmt.Sprintf("> 该项尚未执行（检查项 `%s`）", rid))
			ap("")
			return
		}
		rows := judge(rid, rec.Metrics, spec)
		if len(rows) > 0 {
			ap("| 判定项 | 实测 | 标准 | 结果 |")
			ap("|---|---|---|---|")
			for _, r := range rows {
				ap(fmt.Sprintf("| %s | %s | %s | %s |", r.Label, r.Actual, r.Expected, mark(r.Pass)))
				if r.Pass != nil && !*r.Pass {
					problems = append(problems, fmt.Sprintf("%s: %s 实测 %s，标准 %s", rid, r.Label, r.Actual, r.Expected))
				}
			}
		}
		if rec.Log != "" {
			ap(fmt.Sprintf("\n原始输出: `%s`", rec.Log))
		}
		ap("")
	}

	ap("## 一、开箱核对")
	hw := getResultRec(res, "hw")
	hwM := map[string]interface{}{}
	if hw != nil {
		hwM = hw.Metrics
	}
	if rows, ok := hwM["rows"].([]interface{}); ok && len(rows) > 0 {
		ap("| # | 型号 | 序列号 | UUID | PCI 总线 | 显存 | 驱动 |")
		ap("|---|---|---|---|---|---|---|")
		for _, rv := range rows {
			row, _ := rv.([]interface{})
			cells := []string{}
			for i := 1; i < 8; i++ {
				c := ""
				if i < len(row) {
					c = strings.TrimSpace(str(row[i]))
				}
				if c == "" {
					c = "N/A"
				}
				cells = append(cells, c)
			}
			ap("| " + strings.Join(cells, " | ") + " |")
		}
		ap("")
	} else {
		ap("> 硬件清点未执行")
		ap("")
	}
	if sn := str(hwM["SYSTEM_SN"]); sn != "" {
		ap(fmt.Sprintf("整机序列号: `%s`（用于合同逐项比对与序列号注册）", sn))
	}
	appear := "❌ 未确认"
	if b, _ := man["appearance_ok"].(bool); b {
		appear = "✅ 已确认"
	}
	ap(fmt.Sprintf("外观无运输损伤: %s　合同编号: %s", appear, strOr(man["contract_no"], "—")))
	ap("")

	ap("## 二、配置核对（与配置单逐项比对）")
	judgeTable("cfg")

	ap("## 三、72h 满载烤机")
	judgeTable("burn_status")
	bs := getResultRec(res, "burn_status")
	if bs != nil {
		for _, k := range []string{"ELAPSED_MIN", "DURATION_MIN", "PROGRESS", "ECC_CORR_TOTAL", "ECC_UNCORR_TOTAL", "MON_SAMPLES"} {
			if v, ok := bs.Metrics[k]; ok {
				ap(fmt.Sprintf("- %s: %s", k, str(v)))
			}
		}
		for _, k := range sortedKeys(bs.Metrics) {
			if strings.HasSuffix(k, "_MAX_TEMP") || strings.HasSuffix(k, "_MAX_POWER") {
				ap(fmt.Sprintf("- %s: %s", k, str(bs.Metrics[k])))
			}
		}
		ap("")
	}
	ap("### ECC 日志核查（要求 0 错误）")
	judgeTable("ecc")

	ap("## 四、性能实测（偏差 ≤5%，阈值可在验收标准中调整）")
	ap("### GPU P2P 带宽（强制验收项）")
	judgeTable("p2p")
	ap("### 显存带宽与算力")
	judgeTable("perf")
	ap("### 并发压测（模拟师生访问）")
	judgeTable("conc")

	ap("## 五、软件验收")
	judgeTable("sw")

	ap("## 六、人工确认与文档归档")
	for _, item := range requiredManual {
		ok, _ := man[item.key].(bool)
		s := "❌ 未确认"
		if ok {
			s = "✅"
		}
		ap(fmt.Sprintf("- %s: %s", item.label, s))
		if !ok {
			problems = append(problems, "人工确认未完成: "+item.label)
		}
	}
	ap(fmt.Sprintf("- 出厂测试报告编号: %s", strOr(man["factory_report_no"], "—")))
	ap(fmt.Sprintf("- 备注: %s", strOr(man["notes"], "—")))
	ap("")

	ap("## 验收结论")
	if len(problems) > 0 {
		ap("**存在待解决问题：**")
		for _, p := range problems {
			ap("- ❌ " + p)
		}
	} else {
		ap("**全部检查项通过 ✅（含人工确认项）**")
	}
	ap("")
	ap("| 验收方代表 | 供货方代表 | 日期 |")
	ap("|---|---|---|")
	ap(fmt.Sprintf("| %s |  | %s |", strOr(man["acceptor"], ""), time.Now().Format("2006-01-02")))

	md := strings.Join(L, "\n")
	fname := fmt.Sprintf("验收报告_%s.md", time.Now().Format("20060102_150405"))
	if err := os.WriteFile(filepath.Join(dataDir, fname), []byte(md), 0o644); err != nil {
		return "", "", err
	}
	return fname, md, nil
}

func strOr(v interface{}, def string) string {
	if s := str(v); s != "" {
		return s
	}
	return def
}
