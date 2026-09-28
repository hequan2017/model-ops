// Package store 提供 JSON 文件持久化（读写锁 + 原子落盘）。
package store

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"github.com/hequan2017/model-ops/server/internal/model"
)

// Store 线程安全的文件存储。
type Store struct {
	mu   sync.RWMutex
	path string
	d    model.Data
}

// Open 打开或初始化存储；文件不存在时用 seedFn 生成初始数据。
func Open(path string, seedFn func() model.Data) (*Store, error) {
	s := &Store{path: path}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	b, err := os.ReadFile(path)
	switch {
	case err == nil:
		if err := json.Unmarshal(b, &s.d); err != nil {
			return nil, err
		}
	case os.IsNotExist(err):
		s.d = seedFn()
		if err := s.saveLocked(); err != nil {
			return nil, err
		}
	default:
		return nil, err
	}
	return s, nil
}

// View 只读访问数据快照（持读锁）。
func (s *Store) View(fn func(d *model.Data)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn(&s.d)
}

// Update 修改数据并落盘（持写锁；fn 返回错误则回滚放弃保存）。
func (s *Store) Update(fn func(d *model.Data) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(&s.d); err != nil {
		return err
	}
	return s.saveLocked()
}

// saveLocked 原子写盘（调用方须已持锁）。
func (s *Store) saveLocked() error {
	b, err := json.MarshalIndent(&s.d, "", " ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

// NextID 生成递增 ID（prefix-数字），并立即落盘序号。
func (s *Store) NextID(prefix string, base int) string {
	var id string
	s.Update(func(d *model.Data) error {
		n := d.Seq[prefix]
		if n == 0 {
			n = base
		}
		n++
		d.Seq[prefix] = n
		id = prefix + "-" + itoa(n)
		return nil
	})
	return id
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
