package service_test

import (
	"encoding/json"
	"testing"

	"github.com/Shionyori/edurec-platform/backend/internal/service"
)

func TestBilibiliBvidFromURL(t *testing.T) {
	cases := map[string]string{
		"https://www.bilibili.com/video/BV1DgxCzREbM":         "BV1DgxCzREbM",
		"https://www.bilibili.com/video/BV1DgxCzREbM?p=1":     "BV1DgxCzREbM",
		"https://www.bilibili.com/video/BV1DgxCzREbM/":        "BV1DgxCzREbM",
		"https://example.com/video/BV1DgxCzREbM":              "",
		"https://www.bilibili.com/video/av12345":              "",
		"":                                                    "",
	}

	for input, want := range cases {
		if got := service.BilibiliBvidFromURL(input); got != want {
			t.Errorf("BilibiliBvidFromURL(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestMergeFramesMetadataPreservesExistingFields(t *testing.T) {
	raw := `{"bvid":"BV1xx","duration":"16:08","typename":"校园学习"}`
	frames := map[string]any{
		"sprites": []any{"https://a/1.jpg"},
		"x_len":   float64(10),
		"y_len":   float64(10),
		"count":   float64(100),
	}

	merged, err := service.MergeFramesMetadata(raw, frames)
	if err != nil {
		t.Fatalf("MergeFramesMetadata() error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal([]byte(merged), &got); err != nil {
		t.Fatalf("结果不是合法 JSON: %v", err)
	}
	if got["bvid"] != "BV1xx" || got["duration"] != "16:08" || got["typename"] != "校园学习" {
		t.Errorf("已有字段被破坏: %v", got)
	}
	if _, ok := got["frames"]; !ok {
		t.Error("frames 未写入")
	}
}

func TestMergeFramesMetadataHandlesBadInput(t *testing.T) {
	merged, err := service.MergeFramesMetadata(`{not-json`, map[string]any{"count": 1})
	if err != nil {
		t.Fatalf("坏 metadata 不应报错: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal([]byte(merged), &got); err != nil {
		t.Fatalf("结果不是合法 JSON: %v", err)
	}
	if _, ok := got["frames"]; !ok {
		t.Error("坏输入下仍应写入 frames")
	}
}
