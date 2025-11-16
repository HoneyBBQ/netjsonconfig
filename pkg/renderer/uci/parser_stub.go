package uci

import (
	"context"
	"fmt"

	ast "github.com/honeybbq/netjsonconfig/pkg/ast/uci"
	"github.com/honeybbq/netjsonconfig/pkg/netjsonconfig"
	"github.com/honeybbq/netjsonconfig/pkg/nxerrors"
)

// NotImplementedParser 用于尚未实现解析能力的阶段。
type NotImplementedParser struct{}

func NewNotImplementedParser() *NotImplementedParser {
	return &NotImplementedParser{}
}

func (p *NotImplementedParser) Parse(ctx context.Context, bundle *netjsonconfig.NativeBundle, opts netjsonconfig.ParseOptions) (*ast.Document, error) {
	return nil, nxerrors.New(nxerrors.KindParse, fmt.Errorf("uci parser not implemented"))
}
