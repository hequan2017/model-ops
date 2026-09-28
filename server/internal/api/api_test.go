package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hequan2017/model-ops/server/internal/engine"
	"github.com/hequan2017/model-ops/server/internal/model"
	"github.com/hequan2017/model-ops/server/internal/seed"
	"github.com/hequan2017/model-ops/server/internal/store"
)

func newTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "store.json")
	st, err := store.Open(path, seed.Data)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	mgr := engine.NewManager(st)
	ts := httptest.NewServer(New(st, mgr).Routes())
	t.Cleanup(ts.Close)
	return ts, st
}

func getJSON(t *testing.T, url string, out any) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("GET %s: %v", url, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET %s status=%d", url, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		t.Fatalf("decode %s: %v", url, err)
	}
}

func TestSeedAndOverview(t *testing.T) {
	ts, _ := newTestServer(t)
	var ov Overview
	getJSON(t, ts.URL+"/api/overview", &ov)
	if ov.Orders.Total != 4 {
		t.Fatalf("want 4 seeded orders, got %d", ov.Orders.Total)
	}
	if ov.Installers.Total != 6 {
		t.Fatalf("want 6 installers, got %d", ov.Installers.Total)
	}
	if ov.Orders.ByStatus[model.OrderRework] != 1 {
		t.Fatalf("want 1 rework order, got %d", ov.Orders.ByStatus[model.OrderRework])
	}
	var cats []model.NodeCategory
	getJSON(t, ts.URL+"/api/categories", &cats)
	if len(cats) != 10 {
		t.Fatalf("want 10 categories, got %d", len(cats))
	}
	for _, c := range cats {
		if c.InstallerReq == "" || len(c.Items) == 0 {
			t.Fatalf("category %s missing installer requirement or acceptance items", c.Name)
		}
	}
}

func TestAcceptanceFlow(t *testing.T) {
	ts, _ := newTestServer(t)

	// 生成验收单：TASK-05（教学推理节点满血版，共2台）
	resp, err := http.Post(ts.URL+"/api/tasks/TASK-05/orders", "application/json",
		strings.NewReader(`{"inspectorId":"INS-3"}`))
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("create order: err=%v status=%d", err, resp.StatusCode)
	}
	defer resp.Body.Close()
	var order model.AcceptanceOrder
	if err := json.NewDecoder(resp.Body).Decode(&order); err != nil {
		t.Fatalf("decode order: %v", err)
	}
	if order.Status != model.OrderPending || len(order.Items) != 7 {
		t.Fatalf("unexpected order: status=%s items=%d", order.Status, len(order.Items))
	}

	// 逐项合格
	client := &http.Client{}
	mark := func(key, body string) *model.AcceptanceOrder {
		req, _ := http.NewRequest(http.MethodPut,
			fmt.Sprintf("%s/api/orders/%s/items/%s", ts.URL, order.ID, key),
			strings.NewReader(body))
		r, err := client.Do(req)
		if err != nil || r.StatusCode != http.StatusOK {
			t.Fatalf("update item %s: err=%v status=%d", key, err, r.StatusCode)
		}
		defer r.Body.Close()
		var o model.AcceptanceOrder
		_ = json.NewDecoder(r.Body).Decode(&o)
		return &o
	}
	o := mark("unpack", `{"status":"合格","result":"核对无误","checkedBy":"INS-3"}`)
	if o.Status != model.OrderRunning {
		t.Fatalf("want 进行中 after partial check, got %s", o.Status)
	}
	for _, k := range []string{"config", "burnin", "perf", "conc", "software", "docs"} {
		o = mark(k, fmt.Sprintf(`{"status":"合格","result":"通过","checkedBy":"INS-3"}`))
	}
	if o.Status != model.OrderPassed || o.FinishedAt == "" {
		t.Fatalf("want 已通过 with finishedAt, got %s %q", o.Status, o.FinishedAt)
	}

	// 不合格缺少说明应被拒绝
	req, _ := http.NewRequest(http.MethodPut,
		fmt.Sprintf("%s/api/orders/%s/items/unpack", ts.URL, order.ID),
		strings.NewReader(`{"status":"合格"}`))
	r, _ := client.Do(req)
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("pass without result should be 400, got %d", r.StatusCode)
	}
	r.Body.Close()
}

func TestHTTPStressEngine(t *testing.T) {
	// 真实引擎：对本测试端点做 3 秒并发压测
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("pong"))
	}))
	defer target.Close()

	spec := model.HTTPStressTask{
		Name: "单测并发压测", Target: target.URL, Method: "GET",
		Concurrency: 4, DurationSec: 3, MaxErrRatePct: 1, MaxP95Ms: 500,
	}
	res := engine.RunHTTPStress(spec, nil)
	if res.Total < 100 {
		t.Fatalf("too few requests: %d", res.Total)
	}
	if res.Failed != 0 || !res.Pass {
		t.Fatalf("expected clean pass: %+v", res)
	}
	if res.P95 <= 0 || res.RPS <= 0 {
		t.Fatalf("missing percentiles: %+v", res)
	}
}

func TestBurnInEngine(t *testing.T) {
	spec := model.BurnInTask{
		Name: "单测烤机", GPUs: 2, WattPerGPU: 300,
		DurationSec: 8, IntervalSec: 1, MaxTempC: 85,
	}
	st := engine.NewBurnIn(spec)
	for el := 1; el <= 8; el++ {
		st.Task.Samples = append(st.Task.Samples, st.Sample(el))
	}
	res := st.Finish()
	if !res.Pass || res.EccTotal != 0 {
		t.Fatalf("expected pass: %+v", res)
	}
	if res.MaxTempC < 30 || res.MaxTempC > 85 {
		t.Fatalf("temperature out of range: %v", res.MaxTempC)
	}

	// 注入 ECC 错误应判不合格
	spec.InjectFault = true
	st2 := engine.NewBurnIn(spec)
	for el := 1; el <= 8; el++ {
		st2.Task.Samples = append(st2.Task.Samples, st2.Sample(el))
	}
	if res2 := st2.Finish(); res2.Pass {
		t.Fatalf("injected ECC errors must fail: %+v", res2)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
