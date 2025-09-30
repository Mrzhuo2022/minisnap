package slug

import (
	"crypto/rand"
	"encoding/base32"
	"strings"
)

const (
	slugBytes = 5 // 5 bytes -> 8 chars when base32 (without padding)
)

// New 生成一个短小且URL友好的随机 slug。
func New() string {
	buf := make([]byte, slugBytes)
	_, err := rand.Read(buf)
	if err != nil {
		// 极端情况下 fallback 到固定值，这里返回一个静态 slug。
		return "entry"
	}
	return strings.ToLower(strings.TrimRight(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), "="))
}
