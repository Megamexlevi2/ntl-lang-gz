package native

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"lunex/internal/adaptor"
	"os"
	"path/filepath"
	"runtime"
)

func NativeCacheDir() string {
	return adaptor.NativeCacheDir()
}

func nativeCacheKey(kind string) string {
	h := sha256.New()
	h.Write([]byte(runtime.GOARCH))
	h.Write([]byte(runtime.GOOS))
	h.Write([]byte(kind))
	return hex.EncodeToString(h.Sum(nil))[:16]
}

var mcxMagic = [4]byte{0x4E, 0x4D, 0x43, 0x58}

func LoadNativeCached(kind string) []byte {
	dir := NativeCacheDir()
	path := filepath.Join(dir, nativeCacheKey(kind)+".mcx")
	data, err := os.ReadFile(path)
	if err != nil || len(data) < 8 {
		return nil
	}
	if data[0] != mcxMagic[0] || data[1] != mcxMagic[1] ||
		data[2] != mcxMagic[2] || data[3] != mcxMagic[3] {
		return nil
	}
	expectedSize := binary.LittleEndian.Uint32(data[4:8])
	payload := data[8:]
	if uint32(len(payload)) != expectedSize {
		return nil
	}
	return payload
}

func StoreNativeCached(kind string, code []byte) {
	if len(code) == 0 {
		return
	}
	dir := NativeCacheDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return
	}
	path := filepath.Join(dir, nativeCacheKey(kind)+".mcx")

	hdr := make([]byte, 8)
	copy(hdr[0:4], mcxMagic[:])
	binary.LittleEndian.PutUint32(hdr[4:8], uint32(len(code)))

	_ = os.WriteFile(path, append(hdr, code...), 0600)
}

func ClearNativeCache() int {
	dir := NativeCacheDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	removed := 0
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".mcx" {
			os.Remove(filepath.Join(dir, e.Name()))
			removed++
		}
	}
	return removed
}
