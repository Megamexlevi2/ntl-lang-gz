package runtime

import (
	"fmt"
	"strconv"
	"strings"
)

func encodeURIComponent(s string) string {
	var buf strings.Builder
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' || r == '!' || r == '~' || r == '*' || r == '\'' || r == '(' || r == ')' {
			buf.WriteRune(r)
		} else {
			for _, b := range []byte(string(r)) {
				buf.WriteString(fmt.Sprintf("%%%02X", b))
			}
		}
	}
	return buf.String()
}

func encodeURI(s string) string {
	var buf strings.Builder
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') ||
			r == '-' || r == '_' || r == '.' || r == '!' || r == '~' || r == '*' || r == '\'' || r == '(' || r == ')' ||
			r == ';' || r == ',' || r == '/' || r == '?' || r == ':' || r == '@' || r == '&' ||
			r == '=' || r == '+' || r == '$' || r == '#' {
			buf.WriteRune(r)
		} else {
			for _, b := range []byte(string(r)) {
				buf.WriteString(fmt.Sprintf("%%%02X", b))
			}
		}
	}
	return buf.String()
}

func decodeURIComponent(s string) (string, error) {
	var buf strings.Builder
	for i := 0; i < len(s); {
		if s[i] == '%' && i+2 < len(s) {
			hex := s[i+1 : i+3]
			b, err := strconv.ParseUint(hex, 16, 8)
			if err != nil {
				buf.WriteByte(s[i])
				i++
				continue
			}
			buf.WriteByte(byte(b))
			i += 3
		} else if s[i] == '+' {
			buf.WriteByte(' ')
			i++
		} else {
			buf.WriteByte(s[i])
			i++
		}
	}
	return buf.String(), nil
}

const base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

func base64Encode(data []byte) string {
	var buf strings.Builder
	for i := 0; i < len(data); i += 3 {
		b0 := data[i]
		b1 := byte(0)
		b2 := byte(0)
		if i+1 < len(data) {
			b1 = data[i+1]
		}
		if i+2 < len(data) {
			b2 = data[i+2]
		}
		buf.WriteByte(base64Table[b0>>2])
		buf.WriteByte(base64Table[((b0&3)<<4)|(b1>>4)])
		if i+1 < len(data) {
			buf.WriteByte(base64Table[((b1&0xf)<<2)|(b2>>6)])
		} else {
			buf.WriteByte('=')
		}
		if i+2 < len(data) {
			buf.WriteByte(base64Table[b2&0x3f])
		} else {
			buf.WriteByte('=')
		}
	}
	return buf.String()
}

func base64Decode(s string) ([]byte, error) {
	decode := func(c byte) (byte, bool) {
		switch {
		case c >= 'A' && c <= 'Z':
			return c - 'A', true
		case c >= 'a' && c <= 'z':
			return c - 'a' + 26, true
		case c >= '0' && c <= '9':
			return c - '0' + 52, true
		case c == '+':
			return 62, true
		case c == '/':
			return 63, true
		}
		return 0, false
	}
	var result []byte
	for i := 0; i+3 < len(s); i += 4 {
		b0, ok0 := decode(s[i])
		b1, ok1 := decode(s[i+1])
		if !ok0 || !ok1 {
			continue
		}
		result = append(result, (b0<<2)|(b1>>4))
		if s[i+2] != '=' {
			b2, ok2 := decode(s[i+2])
			if ok2 {
				result = append(result, (b1<<4)|(b2>>2))
			}
		}
		if s[i+3] != '=' {
			b2, _ := decode(s[i+2])
			b3, ok3 := decode(s[i+3])
			if ok3 {
				result = append(result, (b2<<6)|b3)
			}
		}
	}
	return result, nil
}
