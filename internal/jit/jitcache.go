package jit

import (
	"lunex/internal/adaptor"
	"os"
	"path/filepath"
)

func JITCacheDir() string {
	return adaptor.JITCacheDir()
}

func stubPath(key string) string {
	return filepath.Join(JITCacheDir(), key+".bin")
}

func SaveStub(key string, code []byte) error {
	dir := JITCacheDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	return os.WriteFile(stubPath(key), code, 0600)
}

func LoadStub(key string) ([]byte, bool) {
	code, err := os.ReadFile(stubPath(key))
	if err != nil || len(code) == 0 {
		return nil, false
	}
	return code, true
}

func FileJITKey(absPath string) (string, error) {
	return adaptor.MemCacheKey(absPath)
}

func ClearJITCache() (int, error) {
	dir := JITCacheDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, nil
	}
	removed := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".bin" {
			os.Remove(filepath.Join(dir, e.Name()))
			removed++
		}
	}
	return removed, nil
}

func JITCacheInfo() (count int, totalBytes int64) {
	dir := JITCacheDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, 0
	}
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".bin" {
			count++
			if info, err := e.Info(); err == nil {
				totalBytes += info.Size()
			}
		}
	}
	return
}
