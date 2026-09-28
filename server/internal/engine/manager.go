// Manager 统一管理运行中的压测任务：启动、实时快照、完成后写回存储。
package engine

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/hequan2017/model-ops/server/internal/model"
	"github.com/hequan2017/model-ops/server/internal/store"
)

// HTTPRun 运行中的并发压测任务。
type HTTPRun struct {
	Task  model.HTTPStressTask
	Live  *model.HTTPStressResult
	Start time.Time
}

// Manager 压测任务管理器。
type Manager struct {
	st   *store.Store
	mu   sync.RWMutex
	http map[string]*HTTPRun
	burn map[string]*BurnInState
}

// NewManager 创建管理器。
func NewManager(st *store.Store) *Manager {
	return &Manager{st: st, http: map[string]*HTTPRun{}, burn: map[string]*BurnInState{}}
}

// StartHTTP 启动并发压测（异步），返回带 ID 的任务。
func (m *Manager) StartHTTP(spec model.HTTPStressTask) *model.HTTPStressTask {
	spec.Status = model.StressRunning
	spec.CreatedAt = time.Now().Format(time.RFC3339)
	run := &HTTPRun{Task: spec, Start: time.Now()}
	m.mu.Lock()
	m.http[spec.ID] = run
	m.mu.Unlock()

	// 先落库（running 态），结束时更新结果
	_ = m.st.Update(func(d *model.Data) error {
		d.HTTPTests = append([]model.HTTPStressTask{spec}, d.HTTPTests...)
		return nil
	})

	go func() {
		res := RunHTTPStress(spec, func(live *model.HTTPStressResult) {
			m.mu.Lock()
			run.Live = live
			m.mu.Unlock()
		})
		now := time.Now().Format(time.RFC3339)
		_ = m.st.Update(func(d *model.Data) error {
			for i := range d.HTTPTests {
				if d.HTTPTests[i].ID == spec.ID {
					d.HTTPTests[i].Status = model.StressDone
					d.HTTPTests[i].Result = res
					d.HTTPTests[i].FinishedAt = now
					return nil
				}
			}
			return nil
		})
		m.mu.Lock()
		delete(m.http, spec.ID)
		m.mu.Unlock()
		log.Printf("[stress-http] %s finished: total=%d rps=%.1f p95=%.0fms err=%.2f%% pass=%v",
			spec.ID, res.Total, res.RPS, res.P95, res.ErrRatePct, res.Pass)
	}()

	t := spec
	return &t
}

// HTTPLive 查询任务实时状态；第二返回值表示是否在运行中。
func (m *Manager) HTTPLive(id string) (*model.HTTPStressTask, bool) {
	m.mu.RLock()
	run, ok := m.http[id]
	m.mu.RUnlock()
	if ok {
		t := run.Task
		el := int(time.Since(run.Start).Seconds())
		if el < 1 {
			el = 1
		}
		live := run.Live
		if live == nil {
			live = &model.HTTPStressResult{}
		}
		live.DurationSec = el
		t.Result = live
		return &t, true
	}
	var out *model.HTTPStressTask
	m.st.View(func(d *model.Data) {
		for i := range d.HTTPTests {
			if d.HTTPTests[i].ID == id {
				c := d.HTTPTests[i]
				out = &c
			}
		}
	})
	return out, false
}

// StartBurnIn 启动满载烤机（异步，演示模拟采集）。
func (m *Manager) StartBurnIn(spec model.BurnInTask) *model.BurnInTask {
	spec.Status = model.StressRunning
	spec.Mode = "simulated"
	spec.CreatedAt = time.Now().Format(time.RFC3339)
	st := NewBurnIn(spec)
	m.mu.Lock()
	m.burn[spec.ID] = st
	m.mu.Unlock()

	_ = m.st.Update(func(d *model.Data) error {
		d.BurnIns = append([]model.BurnInTask{spec}, d.BurnIns...)
		return nil
	})

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(spec.DurationSec)*time.Second)
		defer cancel()
		tick := time.NewTicker(time.Duration(st.Task.IntervalSec) * time.Second)
		defer tick.Stop()
		for {
			select {
			case <-ctx.Done():
				goto fin
			case <-tick.C:
				el := int(time.Since(st.Start).Seconds())
				m.mu.Lock()
				st.Task.Samples = append(st.Task.Samples, st.Sample(el))
				m.mu.Unlock()
			}
		}
	fin:
		m.mu.Lock()
		// 终止前补一个最终采样点
		el := int(time.Since(st.Start).Seconds())
		st.Task.Samples = append(st.Task.Samples, st.Sample(el))
		res := st.Finish()
		m.mu.Unlock()
		now := time.Now().Format(time.RFC3339)
		_ = m.st.Update(func(d *model.Data) error {
			for i := range d.BurnIns {
				if d.BurnIns[i].ID == spec.ID {
					d.BurnIns[i].Status = model.StressDone
					d.BurnIns[i].Result = res
					d.BurnIns[i].FinishedAt = now
					d.BurnIns[i].Samples = st.Task.Samples
					return nil
				}
			}
			return nil
		})
		m.mu.Lock()
		delete(m.burn, spec.ID)
		m.mu.Unlock()
		log.Printf("[stress-burnin] %s finished: %s", spec.ID, res.Summary)
	}()

	t := spec
	return &t
}

// BurnInLive 查询烤机实时状态；第二返回值表示是否在运行中。
func (m *Manager) BurnInLive(id string) (*model.BurnInTask, bool) {
	m.mu.RLock()
	st, ok := m.burn[id]
	m.mu.RUnlock()
	if ok {
		m.mu.RLock()
		t := st.Task
		el := int(time.Since(st.Start).Seconds())
		t.Samples = append(append([]model.BurnSample(nil), st.Task.Samples...),
			st.peekSample(el))
		t.Message = "运行中（演示模式：模拟遥测采集）"
		m.mu.RUnlock()
		return &t, true
	}
	var out *model.BurnInTask
	m.st.View(func(d *model.Data) {
		for i := range d.BurnIns {
			if d.BurnIns[i].ID == id {
				c := d.BurnIns[i]
				out = &c
			}
		}
	})
	return out, false
}
