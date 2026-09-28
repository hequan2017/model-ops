package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

// ---------------------------------------------------------------------------
// 检查项定义（供界面渲染与执行分发）
// ---------------------------------------------------------------------------
type Check struct {
	ID         string   `json:"id"`
	Cat        string   `json:"cat"`
	Name       string   `json:"name"`
	Desc       string   `json:"desc"`
	Btn        string   `json:"btn"`
	Auto       bool     `json:"auto,omitempty"`
	Inputs     bool     `json:"inputs,omitempty"`
	RowsHeader []string `json:"rows_header,omitempty"`
}

var checks = []Check{
	{ID: "hw", Cat: "① 开箱核对", Name: "硬件清点",
		Desc:       "GPU 型号/序列号/UUID、整机 SN、主板/BIOS、网卡、NVMe 清单（序列号供合同比对与注册）",
		Btn:        "运行硬件清点",
		RowsHeader: []string{"GPU", "型号", "序列号", "UUID", "PCI 总线", "显存", "驱动"}},
	{ID: "cfg", Cat: "② 配置核对", Name: "配置采集与比对",
		Desc: "CPU 核数 / 内存容量 / 磁盘容量 / GPU 数量与型号，自动与下方验收标准比对", Btn: "运行配置核对"},
	{ID: "preflight", Cat: "③ 72h 满载烤机", Name: "环境预检",
		Desc: "烤机前置依赖：tmux/git/nvcc/gcc、GPU 空闲状态、磁盘空间", Btn: "运行预检"},
	{ID: "burn_start", Cat: "③ 72h 满载烤机", Name: "启动烤机",
		Desc: "gpu-burn 满载（时长见验收标准）+ 30 秒采样监控，运行于 tmux 后台，断开 SSH 不中断", Btn: "启动烤机"},
	{ID: "burn_status", Cat: "③ 72h 满载烤机", Name: "烤机状态",
		Desc: "进度、温度/功耗峰值、ECC 计数、Xid 内核日志（可开启 30 秒自动刷新）", Btn: "刷新状态", Auto: true},
	{ID: "burn_stop", Cat: "③ 72h 满载烤机", Name: "停止烤机",
		Desc: "结束负载与监控会话，日志保留在服务器 /root/acceptance/", Btn: "停止烤机"},
	{ID: "ecc", Cat: "③ 72h 满载烤机", Name: "ECC 日志核查",
		Desc: "ECC 模式、可纠正/不可纠正错误计数（验收要求 0 错误）、内核日志", Btn: "核查 ECC",
		RowsHeader: []string{"GPU", "ECC 模式", "可纠正(易失)", "不可纠正(易失)", "可纠正(累计)", "不可纠正(累计)"}},
	{ID: "p2p", Cat: "④ 性能实测", Name: "GPU P2P 带宽（强制验收项）",
		Desc: "nvidia-smi topo + p2pBandwidthLatencyTest（首次自动从 cuda-samples 稀疏克隆构建，需 git+nvcc+make）",
		Btn:  "运行 P2P 测试"},
	{ID: "perf", Cat: "④ 性能实测", Name: "显存带宽与算力实测",
		Desc: "D2D 拷贝带宽、FP32/TF32/FP16/FP8 matmul TFLOPS，与标称值自动比对（阈值见验收标准；解释器在顶部连接设置中配置）",
		Btn:  "运行性能实测"},
	{ID: "conc", Cat: "④ 性能实测", Name: "10 人并发压测",
		Desc: "从本机模拟师生并发访问推理/教学服务，统计 P50/P95/P99 与错误率（压测内网目标请勾选“内网目标”）",
		Btn:  "开始压测", Inputs: true},
	{ID: "sw", Cat: "⑤ 软件验收", Name: "软件环境验证",
		Desc: "驱动/CUDA/nvcc/Docker 与课程镜像、PyTorch 与 TensorFlow GPU 可用性（含真实 GPU 矩阵运算）",
		Btn:  "运行软件验证"},
}

var cats = []string{"① 开箱核对", "② 配置核对", "③ 72h 满载烤机", "④ 性能实测", "⑤ 软件验收"}

var runnable = func() map[string]bool {
	m := map[string]bool{}
	for _, c := range checks {
		m[c.ID] = true
	}
	return m
}()

// ---------------------------------------------------------------------------
// 任务系统
// ---------------------------------------------------------------------------
type Task struct {
	ID      string                 `json:"id"`
	Check   string                 `json:"check"`
	Status  string                 `json:"status"` // running/done/error/stopped
	Output  []string               `json:"-"`
	Metrics map[string]interface{} `json:"metrics"`
	Started time.Time              `json:"-"`
	cmd     *exec.Cmd
	mu      sync.Mutex
}

var (
	tasks   = map[string]*Task{}
	tasksMu sync.RWMutex
)

func newTask(checkID string) *Task {
	b := make([]byte, 4)
	rand.Read(b)
	t := &Task{ID: hex.EncodeToString(b), Check: checkID, Status: "running",
		Metrics: map[string]interface{}{}, Started: time.Now()}
	tasksMu.Lock()
	tasks[t.ID] = t
	tasksMu.Unlock()
	return t
}

func getTask(tid string) *Task {
	tasksMu.RLock()
	defer tasksMu.RUnlock()
	return tasks[tid]
}

func (t *Task) emit(line string) {
	t.mu.Lock()
	t.Output = append(t.Output, line)
	t.mu.Unlock()
}

func (t *Task) parseMetric(line string) {
	body := line[2:]
	if strings.HasPrefix(body, "GPU|") || strings.HasPrefix(body, "ECC|") {
		cells := strings.Split(body, "|")
		t.mu.Lock()
		rows, _ := t.Metrics["rows"].([]interface{})
		row := make([]interface{}, len(cells))
		for i, c := range cells {
			row[i] = c
		}
		t.Metrics["rows"] = append(rows, row)
		t.mu.Unlock()
		return
	}
	if i := strings.Index(body, "="); i > 0 {
		t.mu.Lock()
		t.Metrics[strings.TrimSpace(body[:i])] = strings.TrimSpace(body[i+1:])
		t.mu.Unlock()
	}
}

// ---------------------------------------------------------------------------
// SSH / 进程执行
// ---------------------------------------------------------------------------
func sshArgv(conn Conn, remoteCmd string) []string {
	argv := []string{"-p", strconv.Itoa(conn.Port),
		"-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=accept-new",
		"-o", "ConnectTimeout=10", "-o", "ServerAliveInterval=30"}
	if conn.Key != "" {
		k := conn.Key
		if strings.HasPrefix(k, "~") {
			home, _ := os.UserHomeDir()
			k = filepath.Join(home, k[1:])
		}
		argv = append(argv, "-i", k)
	}
	argv = append(argv, fmt.Sprintf("%s@%s", conn.User, conn.Host))
	if remoteCmd != "" {
		argv = append(argv, remoteCmd)
	}
	return argv
}

func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

func scriptBytes(name string) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(baseDir, "scripts", name))
	if err != nil {
		return nil, err
	}
	return []byte(strings.ReplaceAll(string(b), "\r\n", "\n")), nil
}

// streamProc 启动子进程并把输出逐行送入任务；@@ 开头的行解析为指标。
func streamProc(t *Task, name string, args []string, stdinData []byte) int {
	cmd := exec.Command(name, args...)
	if stdinData != nil {
		cmd.Stdin = strings.NewReader(string(stdinData))
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.emit("[本地异常] " + err.Error())
		return 1
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		t.emit("[本地异常] 启动失败: " + err.Error())
		return 1
	}
	t.mu.Lock()
	t.cmd = cmd
	t.mu.Unlock()

	sc := bufio.NewScanner(stdout)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimRight(sc.Text(), "\r")
		if strings.HasPrefix(line, "@@") {
			t.parseMetric(line)
		}
		t.emit(line)
	}
	err = cmd.Wait()
	t.mu.Lock()
	t.cmd = nil
	t.mu.Unlock()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ExitCode()
		}
		return 1
	}
	return 0
}

func runRemoteScript(t *Task, conn Conn, script string, args []string, envPrefix string) int {
	data, err := scriptBytes(script)
	if err != nil {
		t.emit("[本地异常] 读取脚本失败: " + err.Error())
		return 1
	}
	cmd := envPrefix + "bash -s"
	for _, a := range args {
		cmd += " " + shellQuote(a)
	}
	return streamProc(t, sshBin, sshArgv(conn, cmd), data)
}

func runRemotePython(t *Task, conn Conn, script string) int {
	data, err := scriptBytes(script)
	if err != nil {
		t.emit("[本地异常] 读取脚本失败: " + err.Error())
		return 1
	}
	return streamProc(t, sshBin, sshArgv(conn, conn.RemotePython+" -"), data)
}

func runLocal(t *Task, script string, args []string) int {
	data, err := scriptBytes(script)
	if err != nil {
		t.emit("[本地异常] 读取脚本失败: " + err.Error())
		return 1
	}
	staged := filepath.Join(dataDir, fmt.Sprintf(".stage_%s_%s", t.ID, script))
	if err := os.WriteFile(staged, data, 0o644); err != nil {
		t.emit("[本地异常] 写临时脚本失败: " + err.Error())
		return 1
	}
	defer os.Remove(staged)
	argv := append([]string{staged}, args...)
	py := "python"
	if _, err := exec.LookPath("python"); err != nil {
		py = "python3"
	}
	return streamProc(t, py, argv, nil)
}

// dispatch 启动异步任务
func dispatch(checkID string, params map[string]interface{}) *Task {
	conn := loadConn()
	spec := loadSpec()
	t := newTask(checkID)

	go func() {
		rc := 0
		defer func() {
			if r := recover(); r != nil {
				t.emit(fmt.Sprintf("[本地异常] %v", r))
				if t.Status != "stopped" {
					t.Status = "error"
				}
			}
			saveResult(t)
		}()

		switch checkID {
		case "hw":
			rc = runRemoteScript(t, conn, "01_hardware_inventory.sh", nil, "")
		case "cfg":
			rc = runRemoteScript(t, conn, "02_config_check.sh", nil, "")
		case "preflight":
			rc = runRemoteScript(t, conn, "00_preflight.sh", nil, "")
		case "burn_start":
			rc = runRemoteScript(t, conn, "03_burnin_start.sh",
				[]string{strconv.Itoa(spec.BurnDurationMin)}, "")
		case "burn_status":
			rc = runRemoteScript(t, conn, "03_burnin_status.sh", nil, "")
		case "burn_stop":
			rc = runRemoteScript(t, conn, "03_burnin_stop.sh", nil, "")
		case "ecc":
			rc = runRemoteScript(t, conn, "03_ecc_check.sh", nil, "")
		case "p2p":
			rc = runRemoteScript(t, conn, "04_p2p_bandwidth.sh", nil, "")
		case "perf":
			rc = runRemotePython(t, conn, "04_perf_bench.py")
		case "sw":
			rc = runRemoteScript(t, conn, "05_software_check.sh", nil,
				"env ACC_PY="+shellQuote(conn.RemotePython)+" ")
		case "conc":
			url := strParam(params, "url", conn.ServiceURL)
			if strings.TrimSpace(url) == "" {
				t.emit("[错误] 未配置压测目标 URL")
				t.Status = "error"
				return
			}
			args := []string{url,
				strParam(params, "duration", strconv.Itoa(spec.ConcDurationS)),
				strParam(params, "users", strconv.Itoa(spec.ConcUsers))}
			if body := strings.TrimSpace(strParam(params, "body", "")); body != "" {
				args = append(args, body)
			}
			if boolParam(params, "allow_private") {
				args = append(args, "--allow-private")
			}
			rc = runLocal(t, "04_concurrency_test.py", args)
		default:
			t.emit("[错误] 未知检查项: " + checkID)
			rc = 1
		}
		if t.Status != "stopped" {
			if rc == 0 {
				t.Status = "done"
			} else {
				t.Status = "error"
				t.Metrics["EXIT_CODE"] = strconv.Itoa(rc)
			}
		}
	}()
	return t
}

func strParam(p map[string]interface{}, k, def string) string {
	if v, ok := p[k]; ok {
		switch s := v.(type) {
		case string:
			if s != "" {
				return s
			}
		case float64:
			return strconv.FormatFloat(s, 'f', -1, 64)
		}
	}
	return def
}

func boolParam(p map[string]interface{}, k string) bool {
	v, _ := p[k].(bool)
	return v
}

func (t *Task) stop() {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.cmd != nil && t.cmd.Process != nil {
		_ = t.cmd.Process.Signal(syscall.SIGTERM)
		_ = t.cmd.Process.Kill()
	}
	t.Status = "stopped"
}
