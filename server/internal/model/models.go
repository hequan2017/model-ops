// Package model 定义平台核心数据模型。
//
// 平台聚焦两件事：安装人员要求、设备验收事宜。
// 节点类别仅保留验收流程所需的标识信息（名称/数量/类别），
// 不携带任何设备配置、价格、性能等保密信息。
package model

import "fmt"

// NodeCategory 节点类别：验收单模板与安装人员要求的载体。
type NodeCategory struct {
	ID             string               `json:"id"`
	Code           int                  `json:"code"`
	Name           string               `json:"name"`
	Kind           string               `json:"kind"` // gpu | cpu
	Unit           string               `json:"unit"` // 台 | 组
	Quantity       int                  `json:"quantity"`
	InstallerReq   string               `json:"installerReq"`   // 安装人员要求（原文）
	RequiredSkills []string             `json:"requiredSkills"` // 由要求拆解的技能标签
	Items          []AcceptanceItemTmpl `json:"items"`          // 验收事宜模板
}

// AcceptanceItemTmpl 验收条目模板。
// Category: manual 人工检查 | burnin 满载烤机 | http 并发压测 | performance 性能实测
type AcceptanceItemTmpl struct {
	Key       string `json:"key"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Category  string `json:"category"`
	Mandatory bool   `json:"mandatory"`
}

// Installer 安装人员。Skills 与节点类别的 RequiredSkills 对齐，用于人岗匹配。
type Installer struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Title     string   `json:"title"`
	Level     string   `json:"level"` // 专家级 | 高级 | 中级
	Phone     string   `json:"phone"`
	Skills    []string `json:"skills"`
	Certs     []string `json:"certs"`
	Available bool     `json:"available"`
}

// DeploymentStages 部署通用流程（阶段推进式管理）。
var DeploymentStages = []string{"待启动", "到货清点", "上架安装", "组网接入", "系统与驱动", "压测烤机", "设备验收", "交付使用"}

// DeploymentTask 部署任务：某类节点的一批设备上架部署。
type DeploymentTask struct {
	ID           string   `json:"id"`
	Project      string   `json:"project"`
	CategoryID   string   `json:"categoryId"`
	CategoryName string   `json:"categoryName"`
	Quantity     int      `json:"quantity"`
	Unit         string   `json:"unit"`
	Stage        int      `json:"stage"`
	Status       string   `json:"status"` // 待启动 | 进行中 | 已完成
	InstallerIDs []string `json:"installerIds"`
	StartAt      string   `json:"startAt"`
	Note         string   `json:"note"`
	Accepted     int      `json:"accepted"` // 已生成验收单数量
}

// 验收条目/验收单状态
const (
	ItemPending   = "待检"
	ItemRunning   = "进行中"
	ItemPass      = "合格"
	ItemFail      = "不合格"
	ItemNA        = "不适用"
	OrderPending  = "待验收"
	OrderRunning  = "进行中"
	OrderPassed   = "已通过"
	OrderRework   = "整改中"
	TaskPending   = "待启动"
	TaskRunning   = "进行中"
	TaskDone      = "已完成"
	StressRunning = "running"
	StressDone    = "finished"
)

// AcceptanceItem 验收单条目（从模板实例化）。
type AcceptanceItem struct {
	Key       string `json:"key"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	Category  string `json:"category"`
	Mandatory bool   `json:"mandatory"`
	Status    string `json:"status"`
	Result    string `json:"result"`
	Evidence  string `json:"evidence"`
	CheckedBy string `json:"checkedBy"`
	CheckedAt string `json:"checkedAt"`
}

// AcceptanceOrder 验收单：一台（组）设备的全流程验收记录。
type AcceptanceOrder struct {
	ID           string           `json:"id"`
	TaskID       string           `json:"taskId"`
	CategoryID   string           `json:"categoryId"`
	CategoryName string           `json:"categoryName"`
	DeviceLabel  string           `json:"deviceLabel"`
	InspectorID  string           `json:"inspectorId"`
	Status       string           `json:"status"`
	Items        []AcceptanceItem `json:"items"`
	CreatedAt    string           `json:"createdAt"`
	UpdatedAt    string           `json:"updatedAt"`
	FinishedAt   string           `json:"finishedAt"`
	Conclusion   string           `json:"conclusion"`
}

// HTTPStressTask 校内网并发压测任务（真实压测引擎执行）。
type HTTPStressTask struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Target        string            `json:"target"`
	Method        string            `json:"method"`
	Concurrency   int               `json:"concurrency"`
	DurationSec   int               `json:"durationSec"`
	MaxErrRatePct float64           `json:"maxErrRatePct"`
	MaxP95Ms      float64           `json:"maxP95Ms"`
	AcceptanceID  string            `json:"acceptanceId"` // 可选：关联验收单
	ItemKey       string            `json:"itemKey"`
	Status        string            `json:"status"`
	Message       string            `json:"message"`
	CreatedAt     string            `json:"createdAt"`
	FinishedAt    string            `json:"finishedAt"`
	Result        *HTTPStressResult `json:"result,omitempty"`
}

// SeriesPoint 每秒一个采样点，用于时序图。
type SeriesPoint struct {
	T     int     `json:"t"`
	Count int     `json:"count"`
	AvgMs float64 `json:"avgMs"`
	Errs  int     `json:"errs"`
}

// HTTPStressResult 压测汇总结果。
type HTTPStressResult struct {
	Total       int            `json:"total"`
	Success     int            `json:"success"`
	Failed      int            `json:"failed"`
	RPS         float64        `json:"rps"`
	AvgMs       float64        `json:"avgMs"`
	MinMs       float64        `json:"minMs"`
	MaxMs       float64        `json:"maxMs"`
	P50         float64        `json:"p50"`
	P90         float64        `json:"p90"`
	P95         float64        `json:"p95"`
	P99         float64        `json:"p99"`
	ErrRatePct  float64        `json:"errRatePct"`
	StatusCodes map[string]int `json:"statusCodes"`
	Series      []SeriesPoint  `json:"series"`
	Pass        bool           `json:"pass"`
	DurationSec int            `json:"durationSec"`
}

// BurnInTask 满载烤机任务。
// 注意：本平台运行环境无真实 GPU，采集为演示模式（模拟遥测），
// 判定逻辑（ECC=0、温度不超限）与真实流程一致，可替换为 dcgm-exporter 数据源。
type BurnInTask struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	DeviceLabel  string        `json:"deviceLabel"`
	GPUs         int           `json:"gpus"`
	WattPerGPU   int           `json:"wattPerGpu"`
	DurationSec  int           `json:"durationSec"`
	IntervalSec  int           `json:"intervalSec"`
	MaxTempC     float64       `json:"maxTempC"`
	InjectFault  bool          `json:"injectFault"` // 演示：注入 ECC 错误
	Mode         string        `json:"mode"`        // simulated
	AcceptanceID string        `json:"acceptanceId"`
	ItemKey      string        `json:"itemKey"`
	Status       string        `json:"status"`
	Message      string        `json:"message"`
	CreatedAt    string        `json:"createdAt"`
	FinishedAt   string        `json:"finishedAt"`
	Samples      []BurnSample  `json:"samples"`
	Result       *BurnInResult `json:"result,omitempty"`
}

// BurnSample 单次采样：每 GPU 的温度/功耗/利用率/风扇转速 + 整机 ECC 错误计数。
type BurnSample struct {
	T       int       `json:"t"`
	Temps   []float64 `json:"temps"`
	Powers  []float64 `json:"powers"`
	Utils   []float64 `json:"utils"`
	Fans    []float64 `json:"fans"`
	EccErrs int       `json:"eccErrs"`
}

// BurnInResult 烤机结论：ECC 错误为 0 且最高温度不超限方可通过。
type BurnInResult struct {
	MaxTempC  float64 `json:"maxTempC"`
	AvgPowerW float64 `json:"avgPowerW"`
	EccTotal  int     `json:"eccTotal"`
	Pass      bool    `json:"pass"`
	Summary   string  `json:"summary"`
}

// Data 持久化根结构。
type Data struct {
	Categories []NodeCategory    `json:"categories"`
	Installers []Installer       `json:"installers"`
	Tasks      []DeploymentTask  `json:"tasks"`
	Orders     []AcceptanceOrder `json:"orders"`
	HTTPTests  []HTTPStressTask  `json:"httpTests"`
	BurnIns    []BurnInTask      `json:"burnIns"`
	Seq        map[string]int    `json:"seq"`
}

// NextID 在 Update 事务内生成递增 ID（prefix-数字）；base 为序号从零起步时的起点。
func (d *Data) NextID(prefix string, base int) string {
	n := d.Seq[prefix]
	if n == 0 {
		n = base
	}
	n++
	d.Seq[prefix] = n
	return fmt.Sprintf("%s-%d", prefix, n)
}

// RecalcOrderStatus 依据条目状态重算验收单状态。
func RecalcOrderStatus(items []AcceptanceItem) string {
	hasFail, touched, allDone := false, false, true
	for _, it := range items {
		switch it.Status {
		case ItemFail:
			hasFail, touched = true, true
		case ItemPass, ItemNA:
			touched = true
		case ItemRunning:
			touched = true
			if it.Mandatory {
				allDone = false
			}
		case ItemPending:
			if it.Mandatory {
				allDone = false
			}
		}
	}
	switch {
	case hasFail:
		return OrderRework
	case allDone && touched:
		return OrderPassed
	case touched:
		return OrderRunning
	default:
		return OrderPending
	}
}
