// model-ops 部署管理压测平台后端入口。
//
// 平台核心：安装人员要求管理 与 设备验收事宜（含满载烤机/并发压测执行）。
// 用法：
//
//	go run .                  # 默认 :8080，数据存 ./data/store.json，静态资源 ../web/dist
//	go run . -addr :9000 -data ./data/prod.json -web ../web/dist
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hequan2017/model-ops/server/internal/api"
	"github.com/hequan2017/model-ops/server/internal/engine"
	"github.com/hequan2017/model-ops/server/internal/seed"
	"github.com/hequan2017/model-ops/server/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "监听地址")
	dataPath := flag.String("data", filepath.Join("data", "store.json"), "数据文件路径")
	webDir := flag.String("web", filepath.Join("..", "web", "dist"), "前端静态资源目录（不存在则仅提供 API）")
	flag.Parse()

	st, err := store.Open(*dataPath, seed.Data)
	if err != nil {
		log.Fatalf("打开存储失败: %v", err)
	}

	apiHandler := api.New(st, engine.NewManager(st)).Routes()

	root := http.NewServeMux()
	root.Handle("/api/", apiHandler)
	if info, err := os.Stat(*webDir); err == nil && info.IsDir() {
		root.Handle("/", spaHandler(*webDir))
	} else {
		log.Printf("未找到前端目录 %s，仅提供 API", *webDir)
	}

	srv := &http.Server{
		Addr:              *addr,
		Handler:           root,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	log.Printf("model-ops 部署管理压测平台已启动: http://localhost%s (数据: %s)", *addr, *dataPath)
	log.Fatal(srv.ListenAndServe())
}

// spaHandler 静态资源服务：文件存在则返回，否则回退 index.html（前端路由）。
// 路径解析后必须仍在 root 内，杜绝 ../ 穿越。
func spaHandler(root string) http.Handler {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		log.Fatalf("解析静态目录失败: %v", err)
	}
	indexPath := filepath.Join(absRoot, "index.html")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		abs, err := filepath.Abs(filepath.Join(absRoot, filepath.Clean(r.URL.Path)))
		if err != nil || (abs != absRoot && !strings.HasPrefix(abs, absRoot+string(filepath.Separator))) {
			http.NotFound(w, r)
			return
		}
		if st, err := os.Stat(abs); err != nil || st.IsDir() {
			http.ServeFile(w, r, indexPath)
			return
		}
		http.ServeFile(w, r, abs)
	})
}
