package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	b, _ := json.Marshal(v)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprint(len(b)))
	w.WriteHeader(code)
	w.Write(b)
}

func readBody(r *http.Request) map[string]interface{} {
	body := map[string]interface{}{}
	_ = json.NewDecoder(r.Body).Decode(&body)
	return body
}

func registerRoutes(mux *http.ServeMux) {
	// 页面
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != "/index.html" {
			writeJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		http.ServeFile(w, r, filepath.Join(baseDir, webFS, "index.html"))
	})

	// 状态
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, 200, map[string]interface{}{
			"cats": cats, "checks": checks, "results": loadResults(),
			"spec": loadSpec(), "conn": loadConn(),
		})
	})

	// 任务查询
	mux.HandleFunc("/api/task/", func(w http.ResponseWriter, r *http.Request) {
		tid := strings.TrimPrefix(r.URL.Path, "/api/task/")
		t := getTask(tid)
		if t == nil {
			writeJSON(w, 404, map[string]string{"error": "task not found"})
			return
		}
		t.mu.Lock()
		out := t.Output
		if len(out) > 500 {
			out = out[len(out)-500:]
		}
		snap := map[string]interface{}{
			"id": t.ID, "check": t.Check, "status": t.Status,
			"metrics": t.Metrics, "output": strings.Join(out, "\n"),
		}
		t.mu.Unlock()
		writeJSON(w, 200, snap)
	})

	// 报告
	mux.HandleFunc("/api/report", func(w http.ResponseWriter, r *http.Request) {
		fname, md, err := buildReport()
		if err != nil {
			writeJSON(w, 500, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, 200, map[string]string{"file": fname, "markdown": md})
	})

	// 下载
	mux.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		rel := r.URL.Query().Get("p")
		f := filepath.Join(dataDir, filepath.FromSlash(rel))
		absF, _ := filepath.Abs(f)
		absData, _ := filepath.Abs(dataDir)
		if !strings.HasPrefix(absF, absData+string(os.PathSeparator)) {
			writeJSON(w, 403, map[string]string{"error": "forbidden"})
			return
		}
		st, err := os.Stat(absF)
		if err != nil || st.IsDir() {
			writeJSON(w, 404, map[string]string{"error": "not found"})
			return
		}
		w.Header().Set("Content-Disposition",
			"attachment; filename*=UTF-8''"+url.PathEscape(st.Name()))
		http.ServeFile(w, r, absF)
	})

	// 保存连接
	mux.HandleFunc("/api/conn", func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		conn := loadConn()
		if v, ok := body["host"].(string); ok {
			conn.Host = v
		}
		if v, ok := body["user"].(string); ok {
			conn.User = v
		}
		switch v := body["port"].(type) {
		case float64:
			conn.Port = int(v)
		case string:
			var p int
			if _, err := fmt.Sscanf(v, "%d", &p); err != nil {
				writeJSON(w, 400, map[string]string{"error": "端口必须为数字"})
				return
			}
			conn.Port = p
		}
		if v, ok := body["key"].(string); ok {
			conn.Key = v
		}
		if v, ok := body["remote_python"].(string); ok {
			conn.RemotePython = v
		}
		if v, ok := body["service_url"].(string); ok {
			conn.ServiceURL = v
		}
		saveConn(conn)
		writeJSON(w, 200, map[string]interface{}{"ok": true, "conn": conn})
	})

	// 保存验收标准（并重算判定）
	mux.HandleFunc("/api/spec", func(w http.ResponseWriter, r *http.Request) {
		raw := map[string]interface{}{}
		_ = json.NewDecoder(r.Body).Decode(&raw)
		spec := loadSpec()
		mergeJSON(specF, raw, &spec)
		saveSpec(spec)
		res := loadResults()
		for rid, v := range res {
			if rid == "manual" {
				continue
			}
			if rec := getResultRec(res, rid); rec != nil {
				rec.Judge = judge(rid, rec.Metrics, spec)
				res[rid] = *rec
			}
			_ = v
		}
		saveResults(res)
		writeJSON(w, 200, map[string]interface{}{"ok": true, "spec": spec})
	})

	// 测试 SSH 连接
	mux.HandleFunc("/api/test", func(w http.ResponseWriter, r *http.Request) {
		conn := loadConn()
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, sshBin,
			sshArgv(conn, "echo ACC_OK && hostname && nvidia-smi -L | head -1")...)
		out, err := cmd.CombinedOutput()
		res := map[string]interface{}{"out": strings.TrimSpace(string(out))}
		if ctx.Err() == context.DeadlineExceeded {
			res["ok"] = false
			res["out"] = "SSH 连接超时（25s）"
		} else if err != nil {
			res["ok"] = false
			if ee, ok := err.(*exec.ExitError); ok {
				res["rc"] = ee.ExitCode()
			} else {
				res["out"] = res["out"].(string) + " " + err.Error()
			}
		} else {
			res["ok"] = true
			res["rc"] = 0
		}
		if s := res["out"].(string); len(s) > 2000 {
			res["out"] = s[:2000]
		}
		writeJSON(w, 200, res)
	})

	// 运行检查
	mux.HandleFunc("/api/run", func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		check, _ := body["check"].(string)
		if !runnable[check] {
			writeJSON(w, 400, map[string]string{"error": "未知检查项 " + check})
			return
		}
		params, _ := body["params"].(map[string]interface{})
		t := dispatch(check, params)
		writeJSON(w, 200, map[string]string{"task": t.ID, "check": check})
	})

	// 停止任务
	mux.HandleFunc("/api/stop", func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		tid, _ := body["task"].(string)
		t := getTask(tid)
		if t == nil {
			writeJSON(w, 404, map[string]string{"error": "task not found"})
			return
		}
		t.stop()
		writeJSON(w, 200, map[string]bool{"ok": true})
	})

	// 人工确认
	mux.HandleFunc("/api/manual", func(w http.ResponseWriter, r *http.Request) {
		body := readBody(r)
		allowed := map[string]bool{
			"appearance_ok": true, "contract_no": true, "course_image_ok": true,
			"accounts_ok": true, "factory_report_no": true, "sn_registered": true,
			"warranty_archived": true, "acceptor": true, "notes": true,
		}
		res := loadResults()
		man, _ := res["manual"].(map[string]interface{})
		if man == nil {
			man = map[string]interface{}{}
		}
		for k, v := range body {
			if allowed[k] {
				man[k] = v
			}
		}
		res["manual"] = man
		saveResults(res)
		writeJSON(w, 200, map[string]interface{}{"ok": true, "manual": man})
	})
}

// mergeJSON 用 raw 中的值覆盖已有 spec 文件中对应键（宽松解析数字/字符串）
func mergeJSON(path string, raw map[string]interface{}, out *Spec) {
	// 先基于默认值 + 文件
	b, _ := json.Marshal(out)
	var cur map[string]interface{}
	_ = json.Unmarshal(b, &cur)
	for k, v := range raw {
		if _, ok := cur[k]; !ok {
			continue
		}
		switch cur[k].(type) {
		case float64:
			switch x := v.(type) {
			case float64:
				cur[k] = x
			case string:
				var f float64
				if _, err := fmt.Sscanf(x, "%g", &f); err == nil {
					cur[k] = f
				}
			}
		case string:
			if s, ok := v.(string); ok {
				cur[k] = s
			}
		}
	}
	nb, _ := json.Marshal(cur)
	_ = json.Unmarshal(nb, out)
}
