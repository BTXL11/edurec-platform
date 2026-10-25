package service

import (
	"encoding/json"
	"strings"
)

// MergeFramesMetadata 把关键帧信息合并进资源 metadata，保留已有字段。
//
// 资源 metadata 是任意 JSON 对象（B 站采集会写 bvid/duration/typename 等），
// 这里只覆盖 "frames" 键，其余原样保留；raw 为空或损坏时从空对象开始，
// 不让一次坏数据导致整条资源无法补齐。
func MergeFramesMetadata(raw string, frames map[string]any) (string, error) {
	metadata := map[string]any{}
	if strings.TrimSpace(raw) != "" {
		if err := json.Unmarshal([]byte(raw), &metadata); err != nil {
			metadata = map[string]any{}
		}
	}
	if metadata == nil {
		metadata = map[string]any{}
	}
	metadata["frames"] = frames

	encoded, err := json.Marshal(metadata)
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

// BilibiliBvidFromURL 从 B 站视频链接里取 BV 号：
// https://www.bilibili.com/video/BV1DgxCzREbM?p=1 → BV1DgxCzREbM。
// 取不到（非 B 站链接或格式不符）返回空串。
func BilibiliBvidFromURL(sourceURL string) string {
	sourceURL = strings.TrimSpace(sourceURL)
	if !strings.Contains(sourceURL, "bilibili.com") {
		return ""
	}
	if idx := strings.IndexAny(sourceURL, "?#"); idx >= 0 {
		sourceURL = sourceURL[:idx]
	}
	sourceURL = strings.TrimRight(sourceURL, "/")
	segment := sourceURL[strings.LastIndex(sourceURL, "/")+1:]
	if strings.HasPrefix(segment, "BV") && len(segment) > 2 {
		return segment
	}
	return ""
}
