package meta

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash/adler32"
	"hash/crc32"
	"hash/fnv"
	"os"
	"time"
)

const _lunex_crc uint32 = 0x1850C8D0

const _lunex_fnv uint32 = 0x55D77B0A

const _lunex_adler uint32 = 0x0B7B1D72

const _lunex_anchor = "79cbf28fe7c6eba0"

var _lunex_shown bool

func deriveKey() []byte {
	key := make([]byte, 12)
	for i := 0; i < 3; i++ {
		key[i*4+0] = _lunex_ka[i]
		key[i*4+1] = _lunex_kb[i]
		key[i*4+2] = _lunex_kc[i]
		key[i*4+3] = _lunex_kd[i]
	}
	return key
}

func fnv1a32(data []byte) uint32 {
	h := fnv.New32a()
	h.Write(data)
	return h.Sum32()
}

func validateShards() bool {
	checks := [5]struct {
		shard []byte
		want  uint32
	}{
		{_lunex_s0, _lunex_s0_fnv},
		{_lunex_s1, _lunex_s1_fnv},
		{_lunex_s2, _lunex_s2_fnv},
		{_lunex_s3, _lunex_s3_fnv},
		{_lunex_s4, _lunex_s4_fnv},
	}
	for _, c := range checks {
		if fnv1a32(c.shard) != c.want {
			return false
		}
	}
	return true
}

func ror8(b byte, n uint) byte {
	n &= 7
	return (b >> n) | (b << (8 - n))
}

func decodeProvenance() []byte {
	key := deriveKey()
	combined := make([]byte, 0, 87)
	combined = append(combined, _lunex_s0...)
	combined = append(combined, _lunex_s1...)
	combined = append(combined, _lunex_s2...)
	combined = append(combined, _lunex_s3...)
	combined = append(combined, _lunex_s4...)
	out := make([]byte, len(combined))
	for i, b := range combined {
		out[i] = ror8(b, uint(i%5)+1) ^ key[i%12]
	}
	return out
}

func verifyIntegrity() bool {
	if !validateShards() {
		return false
	}
	decoded := decodeProvenance()

	if crc32.ChecksumIEEE(decoded) != _lunex_crc {
		return false
	}

	if fnv1a32(decoded) != _lunex_fnv {
		return false
	}

	if adler32.Checksum(decoded) != _lunex_adler {
		return false
	}

	sum := sha256.Sum256(decoded)
	anchor := hex.EncodeToString(sum[:])[:16]
	if anchor != _lunex_anchor {
		return false
	}

	return true
}

func verifyAndDisplay() {
	if _lunex_shown {
		return
	}
	if !verifyIntegrity() {
		fmt.Fprintln(os.Stderr,
			"lunex: fatal: build integrity check failed\n"+
				"The binary has been modified or is corrupted.")
		os.Exit(2)
	}
	if len(os.Args) == 1 {
		fmt.Println(string(decodeProvenance()))
		fmt.Println()
		_lunex_shown = true
	}
}

func startIntegrityWatcher() {
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			if !verifyIntegrity() {
				fmt.Fprintln(os.Stderr,
					"lunex: fatal: runtime integrity violation detected")
				os.Exit(2)
			}
		}
	}()
}

func Provenance() string {
	if !verifyIntegrity() {
		fmt.Fprintln(os.Stderr, "lunex: fatal: provenance integrity check failed")
		os.Exit(2)
	}
	return string(decodeProvenance())
}

func Seal() {
	startIntegrityWatcher()
}

func init() {
	verifyAndDisplay()
}
