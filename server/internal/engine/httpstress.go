// Package engine 实现压测引擎。
//
// HTTPStress：真实并发压测引擎——N 个 goroutine 模拟 N 名师生同时访问目标服务，
// 统计 RPS、延迟分位数（P50/P90/P95/P99）、错误率与逐秒时序，用于验收条目
// 「校内网性能与N人并发压测：模拟师生访问，验证响应时间与稳定性」。
package engine

import (
	"context"
	"io"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/hequan2017/model-ops/server/internal/model"
)

// HTTPAgg 并发安全的压测累加器。
type HTTPAgg struct {
	mu        sync.Mutex
	latencies []float64
	total     int
	success   int
	failed    int
	bytes     int64
	codes     map[string]int
	buckets   map[int]*model.SeriesPoint
}

func newHTTPAgg() *HTTPAgg {
	return &HTTPAgg{codes: map[string]int{}, buckets: map[int]*model.SeriesPoint{}}
}

func (a *HTTPAgg) record(sec int, ms float64, ok bool, code string, n int64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if ms < 0 {
		ms = 0
	}
	a.latencies = append(a.latencies, ms)
	a.total++
	if ok {
		a.success++
	} else {
		a.failed++
	}
	a.bytes += n
	a.codes[code]++
	b, exist := a.buckets[sec]
	if !exist {
		b = &model.SeriesPoint{T: sec}
		a.buckets[sec] = b
	}
	b.Count++
	b.AvgMs += (ms - b.AvgMs) / float64(b.Count)
	if !ok {
		b.Errs++
	}
}

// Snapshot 汇总当前结果；percentile 仅在需要时计算。
func (a *HTTPAgg) snapshot(dur int, final bool) *model.HTTPStressResult {
	a.mu.Lock()
	defer a.mu.Unlock()
	r := &model.HTTPStressResult{
		Total:       a.total,
		Success:     a.success,
		Failed:      a.failed,
		StatusCodes: map[string]int{},
	}
	for k, v := range a.codes {
		r.StatusCodes[k] = v
	}
	secs := make([]model.SeriesPoint, 0, len(a.buckets))
	for _, b := range a.buckets {
		secs = append(secs, *b)
	}
	sort.Slice(secs, func(i, j int) bool { return secs[i].T < secs[j].T })
	r.Series = secs
	elapsed := dur
	if elapsed <= 0 {
		elapsed = 1
	}
	r.RPS = round(float64(a.total)/float64(elapsed), 1)
	if a.total > 0 {
		r.ErrRatePct = round(float64(a.failed)*100/float64(a.total), 2)
	}
	if final && len(a.latencies) > 0 {
		lats := append([]float64(nil), a.latencies...)
		sort.Float64s(lats)
		r.MinMs = round(lats[0], 1)
		r.MaxMs = round(lats[len(lats)-1], 1)
		r.P50 = round(pct(lats, 0.50), 1)
		r.P90 = round(pct(lats, 0.90), 1)
		r.P95 = round(pct(lats, 0.95), 1)
		r.P99 = round(pct(lats, 0.99), 1)
		var sum float64
		for _, v := range lats {
			sum += v
		}
		r.AvgMs = round(sum/float64(len(lats)), 1)
	}
	return r
}

func pct(sorted []float64, p float64) float64 {
	idx := int(float64(len(sorted)-1) * p)
	if idx < 0 {
		idx = 0
	}
	return sorted[idx]
}

func round(v float64, n int) float64 {
	p := 1.0
	for i := 0; i < n; i++ {
		p *= 10
	}
	return float64(int64(v*p+0.5)) / p
}

// RunHTTPStress 同步执行一次并发压测并返回完整结果。
// 每秒回调 onTick（可为 nil）用于实时展示。
func RunHTTPStress(spec model.HTTPStressTask, onTick func(*model.HTTPStressResult)) *model.HTTPStressResult {
	agg := newHTTPAgg()
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(spec.DurationSec)*time.Second)
	defer cancel()

	tr := &http.Transport{
		MaxIdleConns:        spec.Concurrency * 2,
		MaxIdleConnsPerHost: spec.Concurrency * 2,
		IdleConnTimeout:     30 * time.Second,
	}
	client := &http.Client{Transport: tr, Timeout: 15 * time.Second}

	start := time.Now()
	method := spec.Method
	if method == "" {
		method = http.MethodGet
	}

	var wg sync.WaitGroup
	for w := 0; w < spec.Concurrency; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if ctx.Err() != nil {
					return
				}
				req, err := http.NewRequestWithContext(ctx, method, spec.Target, nil)
				if err != nil {
					agg.record(int(time.Since(start).Seconds()), 0, false, "ERR", 0)
					return
				}
				t0 := time.Now()
				resp, err := client.Do(req)
				ms := float64(time.Since(t0).Microseconds()) / 1000.0
				if err != nil {
					if ctx.Err() != nil {
						// 压测结束被取消的在途请求不计入统计
						return
					}
					agg.record(int(time.Since(start).Seconds()), ms, false, "ERR", 0)
					continue
				}
				n, _ := io.Copy(io.Discard, resp.Body)
				_ = resp.Body.Close()
				code := http.StatusText(resp.StatusCode)
				if code == "" {
					code = itoaStatus(resp.StatusCode)
				}
				agg.record(int(time.Since(start).Seconds()), ms, resp.StatusCode < 400, code, n)
			}
		}()
	}

	// 逐秒进度回调
	tick := time.NewTicker(time.Second)
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
loop:
	for {
		select {
		case <-done:
			break loop
		case <-tick.C:
			if onTick != nil {
				el := int(time.Since(start).Seconds())
				if el < 1 {
					el = 1
				}
				onTick(agg.snapshot(el, false))
			}
		}
	}
	tick.Stop()

	dur := int(time.Since(start).Seconds())
	if dur < 1 {
		dur = 1
	}
	res := agg.snapshot(dur, true)
	res.DurationSec = dur
	res.Pass = res.ErrRatePct <= spec.MaxErrRatePct && res.P95 <= spec.MaxP95Ms
	return res
}

func itoaStatus(c int) string {
	if c == 0 {
		return "0"
	}
	var buf [4]byte
	i := len(buf)
	for c > 0 {
		i--
		buf[i] = byte('0' + c%10)
		c /= 10
	}
	return string(buf[i:])
}
