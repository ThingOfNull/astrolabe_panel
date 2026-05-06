package i18n

import "testing"

func TestTKnown(t *testing.T) {
	if got := T(CodeBoardNotFound, "zh-CN"); got != "看板不存在" {
		t.Errorf("zh-CN translate = %q", got)
	}
	if got := T(CodeBoardNotFound, "en-US"); got != "Board not found" {
		t.Errorf("en-US translate = %q", got)
	}
}

func TestTFallback(t *testing.T) {
	if got := T(CodeInternal, "fr-FR"); got != "服务器内部错误" {
		t.Errorf("fallback = %q, want zh-CN value", got)
	}
}

func TestTUnknownCode(t *testing.T) {
	if got := T("nonexistent_code", "zh-CN"); got != "nonexistent_code" {
		t.Errorf("unknown code should echo code, got %q", got)
	}
}

func TestRegister(t *testing.T) {
	Register("zh-CN", map[string]string{"custom_key": "自定义"})
	if got := T("custom_key", "zh-CN"); got != "自定义" {
		t.Errorf("registered = %q", got)
	}
}
