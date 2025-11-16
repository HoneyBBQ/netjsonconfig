package netjsonconfig

import "time"

// RenderMode 控制 DSL 渲染策略（如 OpenWrt DSA/legacy）。
type RenderMode int

const (
	// RenderModeAuto 根据输入自动推断。
	RenderModeAuto RenderMode = iota
	// RenderModeDSA 强制 DSA 语法。
	RenderModeDSA
	// RenderModeLegacy 强制 legacy 语法。
	RenderModeLegacy
)

// RenderOptions 控制前向渲染过程。
type RenderOptions struct {
	Mode             RenderMode
	TemplateContext  map[string]any
	Strict           bool
	SkipValidation   bool
	GenerationTag    string
	Timeout          time.Duration
	IncludeAuxiliary bool
}

// ParseOptions 控制反向解析。
type ParseOptions struct {
	Mode           RenderMode
	AllowUnknown   bool
	AssumeTemplate bool
	SkipValidation bool
	Timeout        time.Duration
	SourceMetadata map[string]string
	BestEffort     bool
}
