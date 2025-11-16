package renderer

import (
	"context"

	"github.com/honeybbq/netjsonconfig/pkg/netjsonconfig"
)

// Renderer 定义 DSL 渲染接口，使用泛型约束文档类型。
type Renderer[T any] interface {
	Render(ctx context.Context, doc T, opts netjsonconfig.RenderOptions) (*netjsonconfig.Bundle, error)
}

// Parser 将 DSL 文本解析成领域文档。
type Parser[T any] interface {
	Parse(ctx context.Context, bundle *netjsonconfig.Bundle, opts netjsonconfig.ParseOptions) (T, error)
}
