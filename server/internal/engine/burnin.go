// BurnIn 满载烤机引擎。
//
// 说明：本平台宿主机无真实 GPU，遥测数据为「演示模式」模拟采集
// （温度爬升曲线、按卡功耗、利用率、ECC 错误计数），采样与判定流程
// 与真实烤机一致：满载时长内 ECC 错误累计为 0 且最高温度不超限 → 合格。
// 接入真实环境时，将 sample() 替换为 DCGM-Exporter/nvidia-smi 数据源即可。
package engine

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/hequan2017/model-ops/server/internal/model"
)

// BurnInState 运行中烤机任务的内存态。
type BurnInState struct {
	Task    model.BurnInTask
	Start   time.Time
	mu      sync.Mutex
	rnd     *rand.Rand
	targets []float64 // 每卡稳态温度
	ecc     int
	eccAt   int // 注入故障的时间点（秒），0 表示不注入
}

// NewBurnIn 依据任务规格构造运行态。
func NewBurnIn(t model.BurnInTask) *BurnInState {
	if t.IntervalSec <= 0 {
		t.IntervalSec = 5
	}
	// 限制采样点数量：全程最多约 720 个点，长任务自动拉大间隔
	if minInt := t.DurationSec / 720; t.IntervalSec < minInt {
		t.IntervalSec = minInt
	}
	if t.GPUs <= 0 {
		t.GPUs = 1
	}
	if t.WattPerGPU <= 0 {
		t.WattPerGPU = 300
	}
	st := &BurnInState{Task: t, Start: time.Now(), rnd: rand.New(rand.NewSource(time.Now().UnixNano()))}
	st.targets = make([]float64, t.GPUs)
	for g := range st.targets {
		// 稳态温度 72~83°C，卡间略有差异
		st.targets[g] = 72 + st.rnd.Float64()*11
	}
	if t.InjectFault {
		st.eccAt = t.DurationSec * 6 / 10
	}
	return st
}

// Sample 模拟一次满载采样（改变内部状态）。
func (st *BurnInState) Sample(elapsed int) model.BurnSample {
	st.mu.Lock()
	defer st.mu.Unlock()
	return st.sampleLocked(elapsed)
}

// peekSample 只读快照：不累积 ECC、不推进随机序列。
func (st *BurnInState) peekSample(elapsed int) model.BurnSample {
	st.mu.Lock()
	defer st.mu.Unlock()
	// 用确定性伪噪声代替随机数，避免污染 rnd 序列
	rndSave := st.rnd
	st.rnd = rand.New(rand.NewSource(int64(elapsed) + 1))
	s := st.sampleLocked(elapsed)
	if st.eccAt > 0 && elapsed >= st.eccAt {
		s.EccErrs = st.ecc // 只读当前累计值
	}
	st.rnd = rndSave
	return s
}

func (st *BurnInState) sampleLocked(elapsed int) model.BurnSample {
	n := st.Task.GPUs
	s := model.BurnSample{T: elapsed,
		Temps:  make([]float64, n),
		Powers: make([]float64, n),
		Utils:  make([]float64, n),
		Fans:   make([]float64, n),
	}
	rampSec := 180.0
	if float64(st.Task.DurationSec)*0.05 < rampSec {
		rampSec = math.Max(30, float64(st.Task.DurationSec)*0.05)
	}
	for g := 0; g < n; g++ {
		ramp := math.Min(1, float64(elapsed)/rampSec)
		wob := math.Sin(float64(elapsed)/float64(st.Task.IntervalSec*3)+float64(g)) * 1.2
		temp := 35 + (st.targets[g]-35)*ramp + wob + (st.rnd.Float64()-0.5)*1.4
		if temp < 30 {
			temp = 30
		}
		s.Temps[g] = round(temp, 1)
		s.Powers[g] = round(float64(st.Task.WattPerGPU)*(0.9+st.rnd.Float64()*0.1), 1)
		s.Utils[g] = round(97+st.rnd.Float64()*3, 1)
		s.Fans[g] = round(50+temp*0.4, 1)
	}
	if st.eccAt > 0 && elapsed >= st.eccAt && st.rnd.Float64() < 0.4 {
		st.ecc += 1 + st.rnd.Intn(2)
	}
	s.EccErrs = st.ecc
	return s
}

// Finish 汇总烤机结论。
func (st *BurnInState) Finish() *model.BurnInResult {
	maxTemp := 0.0
	powerSum, powerN := 0.0, 0
	for _, s := range st.Task.Samples {
		for _, t := range s.Temps {
			if t > maxTemp {
				maxTemp = t
			}
		}
		for _, p := range s.Powers {
			powerSum += p
			powerN++
		}
	}
	avg := 0.0
	if powerN > 0 {
		avg = powerSum / float64(powerN)
	}
	pass := st.ecc == 0 && maxTemp <= st.Task.MaxTempC
	hours := st.Task.DurationSec / 3600
	durText := fmt.Sprintf("%d小时", hours)
	if hours < 1 {
		durText = fmt.Sprintf("%d秒（演示时长）", st.Task.DurationSec)
	}
	verdict := "判定合格"
	if !pass {
		verdict = "判定不合格"
	}
	summary := fmt.Sprintf("满载%s完成：最高温度%.1f°C（限值%.0f°C），平均单卡功耗约%.0fW，ECC错误累计%d，%s",
		durText, maxTemp, st.Task.MaxTempC, avg, st.ecc, verdict)
	return &model.BurnInResult{MaxTempC: round(maxTemp, 1), AvgPowerW: round(avg, 1), EccTotal: st.ecc, Pass: pass, Summary: summary}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
