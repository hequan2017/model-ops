// GPU 服务器验收控制台 —— 本地 Web 界面 + SSH 远程执行（Go 版）
//
// 本程序只在本机运行（默认仅监听 127.0.0.1:8765）；所有验收脚本通过 SSH 推送到
// 目标服务器以 `bash -s` / `<python> -` 方式执行，远端不部署任何常驻服务或界面。
// 启动: go run .  或  gpu-acceptance.exe
package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"time"
)

var (
	baseDir  string
	dataDir  string
	logsDir  string
	webFS    = "web"
	resultsF string
	connF    string
	specF    string

	httpPort = envInt("GPU_ACC_WEB_PORT", 8765)
	bindAddr = "127.0.0.1"
	sshBin   = findSSH()
)

func init() {
	exe, err := os.Executable()
	if err != nil {
		baseDir, _ = os.Getwd()
	} else {
		baseDir = filepath.Dir(exe)
	}
	// go run 时 exe 在临时目录，退回当前工作目录
	if _, err := os.Stat(filepath.Join(baseDir, "scripts")); err != nil {
		if wd, e := os.Getwd(); e == nil {
			if _, e2 := os.Stat(filepath.Join(wd, "scripts")); e2 == nil {
				baseDir = wd
			}
		}
	}
	dataDir = filepath.Join(baseDir, "data")
	logsDir = filepath.Join(dataDir, "logs")
	resultsF = filepath.Join(dataDir, "results.json")
	connF = filepath.Join(dataDir, "conn.json")
	specF = filepath.Join(dataDir, "spec.json")
	os.MkdirAll(logsDir, 0o755)
}

func envInt(k string, def int) int {
	if v := os.Getenv(k); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func findSSH() string {
	if p, err := exec.LookPath("ssh"); err == nil {
		return p
	}
	cand := filepath.Join(os.Getenv("SystemRoot"), `System32\OpenSSH\ssh.exe`)
	if _, err := os.Stat(cand); err == nil {
		return cand
	}
	return "ssh"
}

// ---------------------------------------------------------------------------
// JSON 文件读写
// ---------------------------------------------------------------------------
func loadJSON(path string, out interface{}) {
	b, err := os.ReadFile(path)
	if err != nil {
		return
	}
	_ = json.Unmarshal(b, out)
}

func saveJSON(path string, v interface{}) {
	b, _ := json.MarshalIndent(v, "", "  ")
	_ = os.WriteFile(path, b, 0o644)
}

// ---------------------------------------------------------------------------
// 连接配置
// ---------------------------------------------------------------------------
type Conn struct {
	Host         string `json:"host"`
	User         string `json:"user"`
	Port         int    `json:"port"`
	Key          string `json:"key"`
	RemotePython string `json:"remote_python"`
	ServiceURL   string `json:"service_url"`
}

func defaultConn() Conn {
	// 不自动探测本机私钥：仅取显式环境变量，否则留空由界面手动填写
	return Conn{
		Host:         os.Getenv("GPU_ACC_HOST"),
		User:         os.Getenv("GPU_ACC_USER"),
		Port:         envInt("GPU_ACC_PORT", 22),
		Key:          os.Getenv("GPU_ACC_KEY"),
		RemotePython: envStr("GPU_ACC_PY", "python3"),
		ServiceURL:   os.Getenv("GPU_ACC_URL"),
	}
}

func envStr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

var connMu sync.RWMutex

func loadConn() Conn {
	connMu.RLock()
	defer connMu.RUnlock()
	c := defaultConn()
	loadJSON(connF, &c)
	if c.Port == 0 {
		c.Port = 22
	}
	if c.RemotePython == "" {
		c.RemotePython = "python3"
	}
	return c
}

func saveConn(c Conn) {
	connMu.Lock()
	defer connMu.Unlock()
	saveJSON(connF, c)
}

// ---------------------------------------------------------------------------
// 验收标准
// ---------------------------------------------------------------------------
type Spec struct {
	GPUCount         int     `json:"gpu_count"`
	GPUModelContains string  `json:"gpu_model_contains"`
	LogicalCores     int     `json:"logical_cores"`
	MemGB            int     `json:"mem_gb"`
	MemTolPct        int     `json:"mem_tol_pct"`
	DataDiskMount    string  `json:"data_disk_mount"`
	DataDiskMinGB    int     `json:"data_disk_min_gb"`
	BurnDurationMin  int     `json:"burn_duration_min"`
	GPUTempMax       int     `json:"gpu_temp_max"`
	P2PMinGbps       float64 `json:"p2p_min_gbps"`
	MemBwGbps        float64 `json:"mem_bw_gbps"`
	MemBwMinRatio    float64 `json:"mem_bw_min_ratio"`
	FP32TFlops       float64 `json:"fp32_tflops"`
	TF32TFlops       float64 `json:"tf32_tflops"`
	FP16TFlops       float64 `json:"fp16_tflops"`
	FP8TFlops        float64 `json:"fp8_tflops"`
	FlopsMinRatio    float64 `json:"flops_min_ratio"`
	ConcUsers        int     `json:"conc_users"`
	ConcDurationS    int     `json:"conc_duration_s"`
	P95MaxMs         float64 `json:"p95_max_ms"`
	ErrMax           int     `json:"err_max"`
}

func defaultSpec() Spec {
	return Spec{
		GPUCount: 8, GPUModelContains: "L40", LogicalCores: 96, MemGB: 503, MemTolPct: 5,
		DataDiskMount: "/data2", DataDiskMinGB: 3400, BurnDurationMin: 4320, GPUTempMax: 85,
		P2PMinGbps: 20, MemBwGbps: 864, MemBwMinRatio: 0.85,
		FP32TFlops: 90.5, TF32TFlops: 90.5, FP16TFlops: 181, FP8TFlops: 362, FlopsMinRatio: 0.95,
		ConcUsers: 10, ConcDurationS: 120, P95MaxMs: 3000, ErrMax: 0,
	}
}

var specMu sync.RWMutex

func loadSpec() Spec {
	specMu.RLock()
	defer specMu.RUnlock()
	s := defaultSpec()
	loadJSON(specF, &s)
	return s
}

func saveSpec(s Spec) {
	specMu.Lock()
	defer specMu.Unlock()
	saveJSON(specF, s)
}

// ---------------------------------------------------------------------------
// 结果存储
// ---------------------------------------------------------------------------
type JudgeRow struct {
	Label    string `json:"label"`
	Actual   string `json:"actual"`
	Expected string `json:"expected"`
	Pass     *bool  `json:"pass"`
}

type ResultRec struct {
	TS      string                 `json:"ts"`
	Status  string                 `json:"status"`
	Metrics map[string]interface{} `json:"metrics"`
	Log     string                 `json:"log,omitempty"`
	Judge   []JudgeRow             `json:"judge"`
}

type Results map[string]interface{} // rid -> ResultRec | "manual" -> map

var resMu sync.RWMutex

func loadResults() Results {
	resMu.RLock()
	defer resMu.RUnlock()
	r := Results{}
	loadJSON(resultsF, &r)
	return r
}

func saveResults(r Results) {
	resMu.Lock()
	defer resMu.Unlock()
	saveJSON(resultsF, r)
}

func getResultRec(r Results, rid string) *ResultRec {
	v, ok := r[rid]
	if !ok {
		return nil
	}
	b, _ := json.Marshal(v)
	var rec ResultRec
	if json.Unmarshal(b, &rec) != nil {
		return nil
	}
	return &rec
}

func main() {
	mux := http.NewServeMux()
	registerRoutes(mux)
	addr := fmt.Sprintf("%s:%d", bindAddr, httpPort)
	fmt.Printf("GPU 服务器验收控制台已启动: http://%s  （仅监听本机回环，Ctrl+C 退出）\n", addr)
	fmt.Printf("SSH: %s | 数据目录: %s | Go %s\n", sshBin, dataDir, runtime.Version())
	if os.Getenv("GPU_ACC_NO_BROWSER") != "1" {
		go func() {
			time.Sleep(600 * time.Millisecond)
			openBrowser("http://" + addr)
		}()
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fmt.Fprintf(os.Stderr, "监听失败: %v\n", err)
		os.Exit(1)
	}
	if err := http.Serve(ln, mux); err != nil {
		fmt.Fprintf(os.Stderr, "服务退出: %v\n", err)
	}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
