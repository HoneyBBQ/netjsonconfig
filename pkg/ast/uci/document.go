package uci

import commonv1 "github.com/honeybbq/netjson/gen/go/netjson/common/v1"

// Document 表示完整的 UCI 配置集合。
type Document struct {
	Packages []*Package
	Files    []*commonv1.IncludedFile
}

// Package 对应单个 UCI 包（如 network、wireless）。
type Package struct {
	Name     string
	Sections []*Section
}

// Section 是最小 AST 节点。
type Section struct {
	Type    string
	Name    string
	Options map[string][]string
	Lists   map[string][]string
}

// NewSection 创建 Section 并初始化内部 map。
func NewSection(typ, name string) *Section {
	return &Section{
		Type:    typ,
		Name:    name,
		Options: make(map[string][]string),
		Lists:   make(map[string][]string),
	}
}
