// Package api 提供 REST 接口与静态资源服务。
package api

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/hequan2017/model-ops/server/internal/engine"
	"github.com/hequan2017/model-ops/server/internal/model"
	"github.com/hequan2017/model-ops/server/internal/seed"
	"github.com/hequan2017/model-ops/server/internal/store"
)

// API 聚合存储与压测管理器。
type API struct {
	st  *store.Store
	eng *engine.Manager
}

// New 创建 API。
func New(st *store.Store, eng *engine.Manager) *API { return &API{st: st, eng: eng} }

// Routes 注册全部路由。
func (a *API) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/health", a.health)
	mux.HandleFunc("GET /api/overview", a.overview)
	mux.HandleFunc("GET /api/categories", a.categories)
	mux.HandleFunc("GET /api/skills", a.skills)

	mux.HandleFunc("GET /api/installers", a.listInstallers)
	mux.HandleFunc("POST /api/installers", a.createInstaller)
	mux.HandleFunc("PUT /api/installers/{id}", a.updateInstaller)
	mux.HandleFunc("DELETE /api/installers/{id}", a.deleteInstaller)

	mux.HandleFunc("GET /api/tasks", a.listTasks)
	mux.HandleFunc("POST /api/tasks", a.createTask)
	mux.HandleFunc("POST /api/tasks/{id}/stage", a.setTaskStage)
	mux.HandleFunc("POST /api/tasks/{id}/assign", a.assignTask)
	mux.HandleFunc("POST /api/tasks/{id}/orders", a.createOrderFromTask)

	mux.HandleFunc("GET /api/orders", a.listOrders)
	mux.HandleFunc("GET /api/orders/{id}", a.getOrder)
	mux.HandleFunc("PUT /api/orders/{id}", a.updateOrder)
	mux.HandleFunc("PUT /api/orders/{id}/items/{key}", a.updateItem)
	mux.HandleFunc("POST /api/orders/{id}/items/{key}/apply-stress", a.applyStress)

	mux.HandleFunc("GET /api/stress/http", a.listHTTP)
	mux.HandleFunc("POST /api/stress/http", a.startHTTP)
	mux.HandleFunc("GET /api/stress/http/{id}", a.getHTTP)
	mux.HandleFunc("GET /api/stress/http/{id}/live", a.liveHTTP)

	mux.HandleFunc("GET /api/stress/burnin", a.listBurnIn)
	mux.HandleFunc("POST /api/stress/burnin", a.startBurnIn)
	mux.HandleFunc("GET /api/stress/burnin/{id}", a.getBurnIn)
	mux.HandleFunc("GET /api/stress/burnin/{id}/live", a.liveBurnIn)

	return logMW(corsMW(mux))
}

func (a *API) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"status": "ok"})
}

func (a *API) categories(w http.ResponseWriter, _ *http.Request) {
	var out []model.NodeCategory
	a.st.View(func(d *model.Data) { out = d.Categories })
	writeJSON(w, http.StatusOK, out)
}

func (a *API) skills(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, seed.Skills())
}

// ---- 总览 ----

// NodeProgress 单类节点验收进度。
type NodeProgress struct {
	Name     string `json:"name"`
	Quantity int    `json:"quantity"`
	Accepted int    `json:"accepted"`
	Passed   int    `json:"passed"`
}

// Overview 仪表盘聚合数据。
type Overview struct {
	Orders struct {
		Total    int            `json:"total"`
		ByStatus map[string]int `json:"byStatus"`
		PassRate float64        `json:"passRate"`
	} `json:"orders"`
	Items struct {
		Total   int `json:"total"`
		Passed  int `json:"passed"`
		Failed  int `json:"failed"`
		Pending int `json:"pending"`
	} `json:"items"`
	Tasks struct {
		Total   int `json:"total"`
		Running int `json:"running"`
		Done    int `json:"done"`
		Pending int `json:"pending"`
	} `json:"tasks"`
	Installers struct {
		Total   int            `json:"total"`
		ByLevel map[string]int `json:"byLevel"`
		// SkillCover: 技能 → 具备人数
		SkillCover map[string]int `json:"skillCover"`
	} `json:"installers"`
	Stress struct {
		HTTPRunning int `json:"httpRunning"`
		HTTPTotal   int `json:"httpTotal"`
		BurnRunning int `json:"burnRunning"`
		BurnTotal   int `json:"burnTotal"`
	} `json:"stress"`
	NodeProgress []NodeProgress          `json:"nodeProgress"`
	RecentOrders []model.AcceptanceOrder `json:"recentOrders"`
}

func (a *API) overview(w http.ResponseWriter, _ *http.Request) {
	var ov Overview
	ov.Orders.ByStatus = map[string]int{}
	ov.Installers.ByLevel = map[string]int{}
	ov.Installers.SkillCover = map[string]int{}

	a.st.View(func(d *model.Data) {
		progress := map[string]*NodeProgress{}
		for _, c := range d.Categories {
			progress[c.Name] = &NodeProgress{Name: c.Name, Quantity: c.Quantity}
		}
		for _, o := range d.Orders {
			ov.Orders.Total++
			ov.Orders.ByStatus[o.Status]++
			if p, ok := progress[o.CategoryName]; ok {
				p.Accepted++
				if o.Status == model.OrderPassed {
					p.Passed++
				}
			}
			for _, it := range o.Items {
				ov.Items.Total++
				switch it.Status {
				case model.ItemPass:
					ov.Items.Passed++
				case model.ItemFail:
					ov.Items.Failed++
				case model.ItemPending:
					ov.Items.Pending++
				}
			}
		}
		if n := ov.Orders.ByStatus[model.OrderPassed] + ov.Orders.ByStatus[model.OrderRework]; n > 0 {
			ov.Orders.PassRate = round1(float64(ov.Orders.ByStatus[model.OrderPassed]) * 100 / float64(n))
		}
		for _, t := range d.Tasks {
			ov.Tasks.Total++
			switch t.Status {
			case model.TaskRunning:
				ov.Tasks.Running++
			case model.TaskDone:
				ov.Tasks.Done++
			case model.TaskPending:
				ov.Tasks.Pending++
			}
		}
		for _, ins := range d.Installers {
			ov.Installers.Total++
			ov.Installers.ByLevel[ins.Level]++
			for _, s := range ins.Skills {
				ov.Installers.SkillCover[s]++
			}
		}
		for _, t := range d.HTTPTests {
			ov.Stress.HTTPTotal++
			if t.Status == model.StressRunning {
				ov.Stress.HTTPRunning++
			}
		}
		for _, b := range d.BurnIns {
			ov.Stress.BurnTotal++
			if b.Status == model.StressRunning {
				ov.Stress.BurnRunning++
			}
		}
		for _, p := range progress {
			ov.NodeProgress = append(ov.NodeProgress, *p)
		}
		sort.Slice(ov.NodeProgress, func(i, j int) bool { return ov.NodeProgress[i].Name < ov.NodeProgress[j].Name })

		// 最近更新的 6 张验收单
		rec := append([]model.AcceptanceOrder(nil), d.Orders...)
		sort.Slice(rec, func(i, j int) bool { return rec[i].UpdatedAt > rec[j].UpdatedAt })
		if len(rec) > 6 {
			rec = rec[:6]
		}
		ov.RecentOrders = rec
	})
	writeJSON(w, http.StatusOK, ov)
}

func round1(v float64) float64 {
	return float64(int64(v*10+0.5)) / 10
}

// ---- 中间件与工具 ----

// corsOrigins 跨域白名单：默认仅放行本地前端开发服务器；
// 生产同源部署无需跨域。可通过环境变量 MOPS_CORS_ORIGIN 追加（逗号分隔）。
var corsOrigins = func() map[string]bool {
	m := map[string]bool{
		"http://localhost:5173": true,
		"http://127.0.0.1:5173": true,
	}
	if extra := os.Getenv("MOPS_CORS_ORIGIN"); extra != "" {
		for _, o := range strings.Split(extra, ",") {
			m[strings.TrimSpace(o)] = true
		}
	}
	return m
}()

func corsMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && corsOrigins[origin] {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func logMW(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) >= 4 && r.URL.Path[:4] == "/api" {
			log.Printf("%s %s", r.Method, r.URL.Path)
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func fail(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

func decode(r *http.Request, v any) error {
	defer func() { _ = r.Body.Close() }()
	return json.NewDecoder(r.Body).Decode(v)
}
