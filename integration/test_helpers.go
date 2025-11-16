package integration

import (
	"fmt"
	"strings"

	"github.com/honeybbq/netjsonconfig/pkg/netjsonconfig"
)

// bundleToText 将 Bundle 转换为文本格式（用于测试对比）
// UCI 格式：添加 package 行
// 其他格式：直接返回 main config
func bundleToText(bundle *netjsonconfig.Bundle) string {
	if bundle.Metadata.Format == "uci" {
		// UCI 格式：合并所有包并添加 package 行
		var b strings.Builder
		for i, pkg := range bundle.Packages {
			if i > 0 {
				b.WriteString("\n")
			}
			fmt.Fprintf(&b, "package %s\n\n", pkg.Name)
			b.Write(pkg.Content)
		}
		// 保留原样，不修改末尾换行符
		return b.String()
	}

	// 其他格式：返回第一个包的内容
	if len(bundle.Packages) > 0 {
		return string(bundle.Packages[0].Content)
	}
	return ""
}

// normalizeConfig 标准化配置文本用于比较
// 1. 去除首尾空白
// 2. 统一换行符
// 3. 移除空行差异
func normalizeConfig(text string) string {
	// 去除首尾空白
	text = strings.TrimSpace(text)
	// 统一换行符为 \n
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return text
}

// compareConfigs 智能比较配置内容，忽略不重要的空白差异
func compareConfigs(got, want string) bool {
	return normalizeConfig(got) == normalizeConfig(want)
}

// formatConfigDiff 格式化配置差异信息
func formatConfigDiff(got, want string) string {
	gotNorm := normalizeConfig(got)
	wantNorm := normalizeConfig(want)

	if gotNorm == wantNorm {
		return "configs match (after normalization)"
	}

	gotLines := strings.Split(gotNorm, "\n")
	wantLines := strings.Split(wantNorm, "\n")

	var b strings.Builder
	fmt.Fprintf(&b, "config mismatch (got %d lines, want %d lines)\n", len(gotLines), len(wantLines))
	fmt.Fprintf(&b, "--- got (normalized) ---\n%s\n", gotNorm)
	fmt.Fprintf(&b, "--- want (normalized) ---\n%s\n", wantNorm)

	// 逐行比较找出差异
	maxLines := len(gotLines)
	if len(wantLines) > maxLines {
		maxLines = len(wantLines)
	}

	fmt.Fprintf(&b, "--- line-by-line diff ---\n")
	for i := 0; i < maxLines; i++ {
		var gotLine, wantLine string
		if i < len(gotLines) {
			gotLine = gotLines[i]
		}
		if i < len(wantLines) {
			wantLine = wantLines[i]
		}

		if gotLine != wantLine {
			fmt.Fprintf(&b, "Line %d differs:\n", i+1)
			fmt.Fprintf(&b, "  got:  %q\n", gotLine)
			fmt.Fprintf(&b, "  want: %q\n", wantLine)
		}
	}

	return b.String()
}
