// 压测中心处理器：校内网并发压测（真实引擎）与满载烤机（演示模拟采集）。
package api

import (
	"net/http"
	"net/url"

	"github.com/hequan2017/model-ops/server/internal/model"
)

// ---- 校内网并发压测 ----

func (a *API) listHTTP(w http.ResponseWriter, _ *http.Request) {
	var out []model.HTTPStressTask
	a.st.View(func(d *model.Data) { out = d.HTTPTests })
	if out == nil {
		out = []model.HTTPStressTask{}
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) startHTTP(w http.ResponseWriter, r *http.Request) {
	var in model.HTTPStressTask
	if err := decode(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	u, err := url.Parse(in.Target)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		fail(w, http.StatusBadRequest, "目标地址必须是合法的 http(s) URL")
		return
	}
	if in.Concurrency < 1 || in.Concurrency > 2000 {
		fail(w, http.StatusBadRequest, "并发数取值 1~2000（验收规范默认 10 人并发）")
		return
	}
	if in.DurationSec < 1 || in.DurationSec > 86400 {
		fail(w, http.StatusBadRequest, "压测时长取值 1~86400 秒")
		return
	}
	if in.Method == "" {
		in.Method = http.MethodGet
	}
	if in.MaxErrRatePct <= 0 {
		in.MaxErrRatePct = 1
	}
	if in.MaxP95Ms <= 0 {
		in.MaxP95Ms = 500
	}
	// 目标回环保护：仅压本机/内网服务，避免平台被用于攻击外部站点
	if !isAllowedTarget(u) {
		fail(w, http.StatusBadRequest, "目标仅允许本机（127.0.0.1/localhost）或内网地址（10.x/172.16-31.x/192.168.x）")
		return
	}

	var task *model.HTTPStressTask
	if err := a.st.Update(func(d *model.Data) error {
		in.ID = d.NextID("HS", 1000)
		return nil
	}); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	task = a.eng.StartHTTP(in)
	writeJSON(w, http.StatusCreated, task)
}

func (a *API) getHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, _ := a.eng.HTTPLive(id)
	if t == nil {
		fail(w, http.StatusNotFound, "压测记录不存在")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (a *API) liveHTTP(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, _ := a.eng.HTTPLive(id)
	if t == nil {
		fail(w, http.StatusNotFound, "压测记录不存在")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// ---- 满载烤机 ----

func (a *API) listBurnIn(w http.ResponseWriter, _ *http.Request) {
	var out []model.BurnInTask
	a.st.View(func(d *model.Data) {
		for _, b := range d.BurnIns {
			// 列表不携带全部采样点，减小响应体积
			if len(b.Samples) > 60 {
				step := len(b.Samples) / 60
				thin := make([]model.BurnSample, 0, 61)
				for i, s := range b.Samples {
					if i%step == 0 || i == len(b.Samples)-1 {
						thin = append(thin, s)
					}
				}
				b.Samples = thin
			}
			out = append(out, b)
		}
	})
	if out == nil {
		out = []model.BurnInTask{}
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) startBurnIn(w http.ResponseWriter, r *http.Request) {
	var in model.BurnInTask
	if err := decode(r, &in); err != nil {
		fail(w, http.StatusBadRequest, "请求体解析失败: "+err.Error())
		return
	}
	if in.GPUs < 1 || in.GPUs > 16 {
		fail(w, http.StatusBadRequest, "GPU 数量取值 1~16")
		return
	}
	maxDur := 72 * 3600
	if in.DurationSec < 5 || in.DurationSec > maxDur {
		fail(w, http.StatusBadRequest, "烤机时长取值 5~259200 秒（验收规范为 72h，演示可用短时长）")
		return
	}
	if in.IntervalSec < 1 {
		in.IntervalSec = 5
	}
	if in.WattPerGPU <= 0 {
		in.WattPerGPU = 300
	}
	if in.MaxTempC <= 0 {
		in.MaxTempC = 85
	}
	if err := a.st.Update(func(d *model.Data) error {
		in.ID = d.NextID("BI", 1000)
		return nil
	}); err != nil {
		fail(w, http.StatusInternalServerError, err.Error())
		return
	}
	task := a.eng.StartBurnIn(in)
	writeJSON(w, http.StatusCreated, task)
}

func (a *API) getBurnIn(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, _ := a.eng.BurnInLive(id)
	if t == nil {
		fail(w, http.StatusNotFound, "烤机记录不存在")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (a *API) liveBurnIn(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	t, _ := a.eng.BurnInLive(id)
	if t == nil {
		fail(w, http.StatusNotFound, "烤机记录不存在")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// isAllowedTarget 限制压测目标为本机或内网地址（合法内部验收用途）。
func isAllowedTarget(u *url.URL) bool {
	host := u.Hostname()
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	prefixes := []string{"10.", "192.168.", "172.16.", "172.17.", "172.18.", "172.19.",
		"172.20.", "172.21.", "172.22.", "172.23.", "172.24.", "172.25.", "172.26.",
		"172.27.", "172.28.", "172.29.", "172.30.", "172.31."}
	for _, p := range prefixes {
		if len(host) >= len(p) && host[:len(p)] == p {
			return true
		}
	}
	return false
}
