// Package i18n maps RPC error codes to localized text.
//
// zh-CN is complete; others fall back for now.
package i18n

import "sync"

// defaultLocale preference.
const DefaultLocale = "zh-CN"

// Shared RPC error codes; keep stable.
const (
	CodeOK             = "ok"
	CodeInternal       = "internal_error"
	CodeMethodNotFound = "method_not_found"
	CodeInvalidParams  = "invalid_params"
	CodeBoardNotFound  = "board_not_found"
)

var (
	mu    sync.RWMutex
	store = map[string]map[string]string{
		"zh-CN": {
			CodeOK:             "成功",
			CodeInternal:       "服务器内部错误",
			CodeMethodNotFound: "方法不存在",
			CodeInvalidParams:  "参数无效",
			CodeBoardNotFound:  "看板不存在",
		},
		"en-US": {
			CodeOK:             "OK",
			CodeInternal:       "Internal server error",
			CodeMethodNotFound: "Method not found",
			CodeInvalidParams:  "Invalid params",
			CodeBoardNotFound:  "Board not found",
		},
	}
)

// T resolves message for locale with fallbacks.
func T(code, locale string) string {
	mu.RLock()
	defer mu.RUnlock()
	if m, ok := store[locale]; ok {
		if s, ok := m[code]; ok {
			return s
		}
	}
	if locale != DefaultLocale {
		if m, ok := store[DefaultLocale]; ok {
			if s, ok := m[code]; ok {
				return s
			}
		}
	}
	return code
}

// Register merges custom locale tables.
func Register(locale string, entries map[string]string) {
	mu.Lock()
	defer mu.Unlock()
	if store[locale] == nil {
		store[locale] = make(map[string]string, len(entries))
	}
	for k, v := range entries {
		store[locale][k] = v
	}
}
