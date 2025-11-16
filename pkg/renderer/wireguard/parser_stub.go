package wireguard

import (
	"context"

	ast "github.com/honeybbq/netjsonconfig/pkg/ast/wireguard"
	"github.com/honeybbq/netjsonconfig/pkg/netjsonconfig"
	"github.com/honeybbq/netjsonconfig/pkg/nxerrors"
)

type NotImplementedParser struct{}

func NewNotImplementedParser() *NotImplementedParser {
	return &NotImplementedParser{}
}

func (p *NotImplementedParser) Parse(ctx context.Context, bundle *netjsonconfig.NativeBundle, opts netjsonconfig.ParseOptions) (*ast.Document, error) {
	return nil, nxerrors.ErrNotImplemented
}
