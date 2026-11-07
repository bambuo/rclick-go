package i18n

import "testing"

func TestParseLanguage(t *testing.T) {
	cases := map[string]Lang{
		"":           "",
		"system":     "",
		"跟随系统":       "",
		"en":         English,
		"English":    English,
		"zh":         Chinese,
		"zh-Hans":    Chinese,
		"zh-Hans-CN": Chinese,
		"简体中文":       Chinese,
		"fr":         English,
	}
	for tag, want := range cases {
		if got := parse(tag); got != want {
			t.Errorf("parse(%q) = %q, want %q", tag, got, want)
		}
	}
}

func TestChineseTranslations(t *testing.T) {
	Set(Chinese)
	defer Set(English)

	if got := T("Settings…"); got != "设置…" {
		t.Fatalf("T(Settings…) = %q", got)
	}
	if got := T("Quit RClick"); got != "退出 RClick" {
		t.Fatalf("T(Quit RClick) = %q", got)
	}
	if got := ActionName("copy-path", "Copy Path"); got != "复制路径" {
		t.Fatalf("ActionName = %q", got)
	}
	if got := CommandName("qc-zip-file", "Zip File"); got != "压缩文件" {
		t.Fatalf("CommandName = %q", got)
	}
	if got := Sprintf("Save failed: %s", "disk full"); got != "保存失败：disk full" {
		t.Fatalf("Sprintf = %q", got)
	}
}

func TestEnglishIsIdentity(t *testing.T) {
	Set(English)
	if got := T("Settings…"); got != "Settings…" {
		t.Fatalf("T(Settings…) = %q", got)
	}
	if got := ActionName("copy-path", "Copy Path"); got != "Copy Path" {
		t.Fatalf("ActionName = %q", got)
	}
}
