package openvpn

import (
	"context"

	ast "github.com/honeybbq/netjsonconfig/pkg/ast/openvpn"
	"github.com/honeybbq/netjsonconfig/pkg/netjsonconfig"
	"github.com/honeybbq/netjsonconfig/pkg/nxerrors"
)

// NotImplementedParser 目前仅占位，Future: 解析 OpenVPN 配置。
type NotImplementedParser struct{}

func NewNotImplementedParser() *NotImplementedParser {
	return &NotImplementedParser{}
}

func (p *NotImplementedParser) Parse(ctx context.Context, bundle *netjsonconfig.NativeBundle, opts netjsonconfig.ParseOptions) (*ast.Document, error) {
	return nil, nxerrors.ErrNotImplemented
}
