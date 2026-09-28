// Package seed 提供初始数据：节点类别（仅标识+安装人员要求+验收事宜模板）、
// 安装人员、部署任务与示例验收单/压测记录。
package seed

import (
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/hequan2017/model-ops/server/internal/model"
)

// 安装人员要求原文（GPU 类节点通用）
const gpuInstallerReq = "具备NVIDIA专业GPU服务器部署经验，熟悉GPU推理服务部署（vLLM/Triton）与DCGM-Exporter监控配置，熟悉CUDA/TensorRT环境配置与集群调度搭建；掌握MIG多实例配置与ECC日志读取。"

// 安装人员要求原文（CPU 仿真节点）
const cpuInstallerReq = "熟悉高性能计算集群部署，掌握oneAPI/MKL数学库配置与CAE软件许可管理；具备内存全通道均衡安装（32×32GB）与SPEC基准测试能力。"

// 验收事宜原文（GPU 类节点通用），拆为六个条目
func gpuItems() []model.AcceptanceItemTmpl {
	return []model.AcceptanceItemTmpl{
		{Key: "unpack", Title: "① 开箱核对", Content: "显卡型号/序列号与合同逐项比对，外观无运输损伤", Category: "manual", Mandatory: true},
		{Key: "config", Title: "② 配置核对", Content: "CPU核数、内存容量、SSD容量与配置单逐项比对", Category: "manual", Mandatory: true},
		{Key: "burnin", Title: "③ 72h满载烤机", Content: "运行GPU压测监测温度/功耗/稳定性，ECC事件日志确认0错误", Category: "burnin", Mandatory: true},
		{Key: "perf", Title: "④ 性能实测", Content: "GPU P2P带宽（强制验收项）、显存带宽、FP32/FP8算力与标称值比对（偏差≤5%）", Category: "performance", Mandatory: true},
		{Key: "conc", Title: "⑤ 校内网并发压测", Content: "校内网性能与10人并发压测：模拟师生访问，验证响应时间与稳定性", Category: "http", Mandatory: true},
		{Key: "software", Title: "⑥ 软件验收", Content: "CUDA驱动、PyTorch/TensorFlow框架安装运行验证；课程镜像部署、师生账号开通验证", Category: "manual", Mandatory: true},
		{Key: "docs", Title: "⑦ 文档归档", Content: "出厂测试报告、序列号注册、质保凭证归档", Category: "manual", Mandatory: false},
	}
}

// 验收事宜原文（CPU 仿真节点），五个条目
func cpuItems() []model.AcceptanceItemTmpl {
	return []model.AcceptanceItemTmpl{
		{Key: "config", Title: "① 配置核对", Content: "CPU核数/线程数、内存通道数与配置单比对", Category: "manual", Mandatory: true},
		{Key: "spec", Title: "② SPEC CPU 2017基准测试", Content: "整数/浮点得分与行业基准比对", Category: "performance", Mandatory: true},
		{Key: "case", Title: "③ 真实工程算例计时", Content: "使用Fluent/Abaqus标准算例验收实际求解性能", Category: "performance", Mandatory: true},
		{Key: "burnin", Title: "④ 稳定性测试与并发压测", Content: "长时间满载运行监测；校内网性能与3人并发压测：模拟师生访问，验证响应时间与稳定性", Category: "burnin", Mandatory: true},
		{Key: "docs", Title: "⑤ 文档归档", Content: "基准测试报告归档", Category: "manual", Mandatory: false},
	}
}

// GPU 类节点要求的技能标签（由安装人员要求拆解）
var gpuSkills = []string{
	"NVIDIA专业GPU服务器部署", "vLLM/Triton推理服务部署", "DCGM-Exporter监控配置",
	"CUDA/TensorRT环境配置", "集群调度搭建", "MIG多实例配置", "ECC日志读取",
}

// CPU 仿真节点要求的技能标签
var cpuSkills = []string{
	"高性能计算集群部署", "oneAPI/MKL数学库配置", "CAE软件许可管理", "内存全通道均衡安装", "SPEC基准测试",
}

func cat(code int, name, kind, unit string, qty int) model.NodeCategory {
	c := model.NodeCategory{
		ID: fmt.Sprintf("NODE-%02d", code), Code: code, Name: name,
		Kind: kind, Unit: unit, Quantity: qty,
	}
	if kind == "cpu" {
		c.InstallerReq = cpuInstallerReq
		c.RequiredSkills = cpuSkills
		c.Items = cpuItems()
	} else {
		c.InstallerReq = gpuInstallerReq
		c.RequiredSkills = gpuSkills
		c.Items = gpuItems()
	}
	return c
}

// Skills 技能清单（按类别分组）。
func Skills() map[string][]string {
	return map[string][]string{"gpu": gpuSkills, "cpu": cpuSkills}
}

// Categories 全部节点类别。
func Categories() []model.NodeCategory {
	return []model.NodeCategory{
		cat(1, "科研微调节点（满血版）", "gpu", "台", 1),
		cat(2, "科研微调节点（标准版）", "gpu", "台", 2),
		cat(3, "科研推理节点（满血版）", "gpu", "台", 1),
		cat(4, "科研推理节点（标准版）", "gpu", "台", 1),
		cat(5, "教学推理节点（满血版）", "gpu", "台", 2),
		cat(6, "教学推理节点（高级版）", "gpu", "台", 2),
		cat(7, "教学推理节点（标准版）", "gpu", "台", 4),
		cat(8, "CPU仿真节点", "cpu", "台", 1),
		cat(9, "智能体集群", "gpu", "组", 5),
		cat(10, "实训集群", "gpu", "组", 20),
	}
}

func installer(id, name, title, level, phone string, skills, certs []string, avail bool) model.Installer {
	return model.Installer{ID: id, Name: name, Title: title, Level: level, Phone: phone,
		Skills: skills, Certs: certs, Available: avail}
}

// Installers 初始安装人员名册。
func Installers() []model.Installer {
	return []model.Installer{
		installer("INS-1", "张伟", "GPU集群部署专家", "专家级", "138-0000-0001",
			[]string{"NVIDIA专业GPU服务器部署", "vLLM/Triton推理服务部署", "DCGM-Exporter监控配置", "CUDA/TensorRT环境配置", "集群调度搭建", "MIG多实例配置", "ECC日志读取"},
			[]string{"NVIDIA企业级部署认证", "DCGM运维认证"}, true),
		installer("INS-2", "李强", "高级GPU系统工程师", "高级", "138-0000-0002",
			[]string{"NVIDIA专业GPU服务器部署", "vLLM/Triton推理服务部署", "CUDA/TensorRT环境配置", "MIG多实例配置", "ECC日志读取"},
			[]string{"NVIDIA企业级部署认证"}, true),
		installer("INS-3", "王芳", "高级运维工程师", "高级", "138-0000-0003",
			[]string{"NVIDIA专业GPU服务器部署", "DCGM-Exporter监控配置", "CUDA/TensorRT环境配置", "集群调度搭建", "MIG多实例配置"},
			[]string{"DCGM运维认证"}, true),
		installer("INS-4", "赵敏", "推理服务工程师", "中级", "138-0000-0004",
			[]string{"vLLM/Triton推理服务部署", "CUDA/TensorRT环境配置", "ECC日志读取"},
			nil, true),
		installer("INS-5", "陈杰", "HPC集群专家", "专家级", "138-0000-0005",
			[]string{"高性能计算集群部署", "oneAPI/MKL数学库配置", "CAE软件许可管理", "内存全通道均衡安装", "SPEC基准测试"},
			[]string{"SPEC基准测试认证"}, false),
		installer("INS-6", "刘洋", "部署工程师", "中级", "138-0000-0006",
			[]string{"NVIDIA专业GPU服务器部署", "DCGM-Exporter监控配置", "集群调度搭建"},
			nil, true),
	}
}

// Data 构造完整初始数据（含示例验收单与压测历史）。
func Data() model.Data {
	now := time.Now()
	day := func(d int) string { return now.AddDate(0, 0, d).Format(time.RFC3339) }

	cats := Categories()
	byID := map[string]model.NodeCategory{}
	for _, c := range cats {
		byID[c.ID] = c
	}

	// 部署任务：一期建设，各节点类别一项，阶段各异
	tasks := []model.DeploymentTask{}
	taskStage := map[int]int{1: 6, 2: 5, 3: 4, 4: 3, 5: 2, 6: 1, 7: 0, 8: 5, 9: 1, 10: 0}
	taskInstallers := map[int][]string{1: {"INS-1", "INS-2"}, 2: {"INS-2", "INS-3"}, 8: {"INS-5"}}
	for _, c := range cats {
		st := taskStage[c.Code]
		status := model.TaskRunning
		if st == 0 {
			status = model.TaskPending
		} else if st >= len(model.DeploymentStages)-2 {
			status = model.TaskRunning
		}
		tasks = append(tasks, model.DeploymentTask{
			ID: fmt.Sprintf("TASK-%02d", c.Code), Project: "AI边缘算力中心一期建设",
			CategoryID: c.ID, CategoryName: c.Name, Quantity: c.Quantity, Unit: c.Unit,
			Stage: st, Status: status, InstallerIDs: taskInstallers[c.Code],
			StartAt: day(-20 + c.Code), Note: "",
		})
	}

	// 示例验收单：三类典型状态
	newOrder := func(id, taskID string, c model.NodeCategory, seq int, inspector string) model.AcceptanceOrder {
		items := make([]model.AcceptanceItem, 0, len(c.Items))
		for _, t := range c.Items {
			items = append(items, model.AcceptanceItem{Key: t.Key, Title: t.Title, Content: t.Content,
				Category: t.Category, Mandatory: t.Mandatory, Status: model.ItemPending})
		}
		return model.AcceptanceOrder{
			ID: id, TaskID: taskID, CategoryID: c.ID, CategoryName: c.Name,
			DeviceLabel: fmt.Sprintf("%s #%d", c.Name, seq), InspectorID: inspector,
			Status: model.OrderPending, Items: items, CreatedAt: day(-10), UpdatedAt: day(-10),
		}
	}
	mark := func(o *model.AcceptanceOrder, key, status, result, by string) {
		for i := range o.Items {
			if o.Items[i].Key == key {
				o.Items[i].Status = status
				o.Items[i].Result = result
				o.Items[i].CheckedBy = by
				o.Items[i].CheckedAt = day(-8)
			}
		}
		o.UpdatedAt = day(-8)
	}

	orders := []model.AcceptanceOrder{}

	// 1) 已通过：科研微调节点（满血版）#1 全条目合格
	o1 := newOrder("ACC-1001", "TASK-01", byID["NODE-01"], 1, "INS-1")
	for _, k := range []string{"unpack", "config", "burnin", "perf", "conc", "software", "docs"} {
		res := map[string]string{
			"unpack":   "8张显卡序列号与合同一致，外观无损伤",
			"config":   "CPU/内存/SSD 与配置单逐项一致",
			"burnin":   "72h满载完成，最高温度82.4°C，ECC错误0（烤机记录 BI-1001）",
			"perf":     "P2P带宽实测达标，FP32/FP8 偏差 2.1%/1.7%（≤5%）",
			"conc":     "10并发错误率0.2%，P95 210ms（压测记录 HS-1001）",
			"software": "CUDA驱动与框架验证通过，课程镜像与账号开通",
			"docs":     "出厂报告/序列号注册/质保凭证已归档",
		}[k]
		mark(&o1, k, model.ItemPass, res, "INS-1")
	}
	o1.Status = model.OrderPassed
	o1.FinishedAt = day(-6)
	o1.Conclusion = "全部条目验收合格，同意交付。"
	orders = append(orders, o1)

	// 2) 整改中：科研微调节点（标准版）#1 性能实测不合格
	o2 := newOrder("ACC-1002", "TASK-02", byID["NODE-02"], 1, "INS-2")
	mark(&o2, "unpack", model.ItemPass, "序列号与合同一致", "INS-2")
	mark(&o2, "config", model.ItemPass, "配置与配置单一致", "INS-2")
	mark(&o2, "burnin", model.ItemPass, "72h满载完成，ECC错误0（烤机记录 BI-1002）", "INS-2")
	mark(&o2, "perf", model.ItemFail, "GPU P2P带宽实测仅为标称值的78%，不满足偏差≤5%要求，已提请厂商整改", "INS-2")
	o2.Status = model.OrderRework
	orders = append(orders, o2)

	// 3) 进行中：科研推理节点（满血版）#1
	o3 := newOrder("ACC-1003", "TASK-03", byID["NODE-03"], 1, "INS-1")
	mark(&o3, "unpack", model.ItemPass, "开箱核对无误", "INS-1")
	mark(&o3, "config", model.ItemPass, "配置核对一致", "INS-1")
	o3.Status = model.OrderRunning
	orders = append(orders, o3)

	// 4) 待验收：CPU仿真节点 #1
	o4 := newOrder("ACC-1004", "TASK-08", byID["NODE-08"], 1, "INS-5")
	orders = append(orders, o4)

	for i := range orders {
		orders[i].CreatedAt = day(-10 + i)
		orders[i].UpdatedAt = day(-7 + i)
	}
	tasks[0].Accepted = 1 // TASK-01
	tasks[1].Accepted = 1 // TASK-02
	tasks[2].Accepted = 1 // TASK-03
	tasks[7].Accepted = 1 // TASK-08

	// 压测历史：并发压测两条 + 烤机两条
	httpTests := []model.HTTPStressTask{
		{
			ID: "HS-1001", Name: "满血版推理节点 10人并发验收压测", Target: "http://127.0.0.1:8080/api/health",
			Method: "GET", Concurrency: 10, DurationSec: 60, MaxErrRatePct: 1, MaxP95Ms: 500,
			AcceptanceID: "ACC-1001", ItemKey: "conc", Status: model.StressDone,
			CreatedAt: day(-7), FinishedAt: day(-7),
			Result: genHTTPResult(10, 60, 0.2, 210, true),
		},
		{
			ID: "HS-1002", Name: "标准版节点 10人并发预检", Target: "http://127.0.0.1:8080/api/health",
			Method: "GET", Concurrency: 10, DurationSec: 30, MaxErrRatePct: 1, MaxP95Ms: 500,
			Status: model.StressDone, CreatedAt: day(-5), FinishedAt: day(-5),
			Result: genHTTPResult(10, 30, 0.1, 180, true),
		},
	}

	burnIns := []model.BurnInTask{
		{
			ID: "BI-1001", Name: "满血版 72h满载烤机", DeviceLabel: "科研微调节点（满血版） #1",
			GPUs: 8, WattPerGPU: 600, DurationSec: 72 * 3600, IntervalSec: 3600, MaxTempC: 85,
			Mode: "simulated", AcceptanceID: "ACC-1001", ItemKey: "burnin",
			Status: model.StressDone, CreatedAt: day(-12), FinishedAt: day(-9),
			Samples: genBurnSamples(8, 72*3600, 3600, 600, false, 1),
			Result:  genBurnResult(82.4, 8, 600, 0, 72*3600, true),
		},
		{
			ID: "BI-1002", Name: "标准版 72h满载烤机", DeviceLabel: "科研微调节点（标准版） #1",
			GPUs: 8, WattPerGPU: 600, DurationSec: 72 * 3600, IntervalSec: 3600, MaxTempC: 85,
			Mode: "simulated", AcceptanceID: "ACC-1002", ItemKey: "burnin",
			Status: model.StressDone, CreatedAt: day(-11), FinishedAt: day(-8),
			Samples: genBurnSamples(8, 72*3600, 3600, 600, false, 2),
			Result:  genBurnResult(79.8, 8, 600, 0, 72*3600, true),
		},
	}

	return model.Data{
		Categories: cats,
		Installers: Installers(),
		Tasks:      tasks,
		Orders:     orders,
		HTTPTests:  httpTests,
		BurnIns:    burnIns,
		Seq:        map[string]int{"INS": 6, "TASK": 10, "ACC": 1004, "HS": 1002, "BI": 1002},
	}
}

// ---- 示例压测数据生成（仅用于历史记录种子） ----

func genHTTPResult(conc, dur int, errRate, p95 float64, pass bool) *model.HTTPStressResult {
	r := rand.New(rand.NewSource(42))
	total := conc * dur * 38
	failed := int(float64(total) * errRate / 100)
	series := make([]model.SeriesPoint, 0, dur)
	for s := 1; s <= dur; s++ {
		cnt := 36 + r.Intn(6)
		avg := 150 + float64(r.Intn(40))
		series = append(series, model.SeriesPoint{T: s, Count: cnt, AvgMs: math.Round(avg*10) / 10, Errs: r.Intn(100) / 60})
	}
	return &model.HTTPStressResult{
		Total: total, Success: total - failed, Failed: failed,
		RPS: math.Round(float64(total)/float64(dur)*10) / 10, AvgMs: 168, MinMs: 42, MaxMs: 640,
		P50: 155, P90: 196, P95: p95, P99: 312,
		ErrRatePct: errRate, StatusCodes: map[string]int{"200": total - failed, "500": failed},
		Series: series, Pass: pass, DurationSec: dur,
	}
}

func genBurnSamples(gpus, dur, interval, watt int, fault bool, seed int64) []model.BurnSample {
	r := rand.New(rand.NewSource(seed))
	n := dur/interval + 1
	out := make([]model.BurnSample, 0, n)
	targets := make([]float64, gpus)
	for g := range targets {
		targets[g] = 72 + float64(g%5)*2.5
	}
	for i := 0; i < n; i++ {
		t := i * interval
		s := model.BurnSample{T: t, Temps: make([]float64, gpus), Powers: make([]float64, gpus),
			Utils: make([]float64, gpus), Fans: make([]float64, gpus)}
		for g := 0; g < gpus; g++ {
			ramp := math.Min(1, float64(t)/1800)
			wob := math.Sin(float64(t)/600+float64(g)) * 1.2
			temp := 35 + (targets[g]-35)*ramp + wob + (r.Float64()-0.5)*1.4
			s.Temps[g] = math.Round(temp*10) / 10
			s.Powers[g] = math.Round((float64(watt)*(0.9+r.Float64()*0.1))*10) / 10
			s.Utils[g] = 97 + r.Float64()*3
			s.Fans[g] = 55 + temp*0.35
		}
		if fault && t > dur*6/10 {
			s.EccErrs = r.Intn(3)
		}
		out = append(out, s)
	}
	return out
}

func genBurnResult(maxTemp float64, gpus, watt, ecc, dur int, pass bool) *model.BurnInResult {
	summary := fmt.Sprintf("满载%d完成：最高温度%.1f°C（限值85°C），平均单卡功耗约%dW，ECC错误累计%d，%s",
		dur/3600, maxTemp, watt, ecc, map[bool]string{true: "判定合格", false: "判定不合格"}[pass])
	return &model.BurnInResult{MaxTempC: maxTemp, AvgPowerW: float64(watt), EccTotal: ecc, Pass: pass, Summary: summary}
}
