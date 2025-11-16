package nxerrors

import (
	"errors"
	"fmt"
)

// Kind identifies the high level class of an error surfaced by netjsonconfig-go.
type Kind string

const (
	// KindValidation indicates user supplied NetJSON data failed validation.
	KindValidation Kind = "validation"
	// KindParse indicates native configuration无法解析。
	KindParse Kind = "parse"
	// KindRender indicates DSL 渲染失败。
	KindRender Kind = "render"
	// KindConflict 表示同步或版本冲突。
	KindConflict Kind = "conflict"
	// KindUnsupported 表示暂不支持的功能。
	KindUnsupported Kind = "unsupported"
	// KindInternal 表示未知或内部错误。
	KindInternal Kind = "internal"
)

// Error 包装底层错误并附加 Kind，方便调用方根据类型处理。
type Error struct {
	Kind Kind
	Err  error
}

// Error 实现 error 接口。
func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err == nil {
		return string(e.Kind)
	}
	return fmt.Sprintf("%s: %v", e.Kind, e.Err)
}

// Unwrap 允许 errors.Is/As 访问底层错误。
func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// New 创建指定 Kind 的错误。
func New(kind Kind, err error) error {
	if err == nil {
		err = errors.New(string(kind))
	}
	return &Error{Kind: kind, Err: err}
}

var (
	// ErrNotImplemented 统一指示功能尚未实现。
	ErrNotImplemented = errors.New("netjsonconfig: not implemented")
)
