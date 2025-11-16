package vxlan

import (
	"context"
	"errors"

	vxlanv1 "github.com/honeybbq/netjson/gen/go/netjson/vxlan/v1"

	"google.golang.org/protobuf/proto"

	domain "github.com/honeybbq/netjsonconfig/domain/vxlan"
	ast "github.com/honeybbq/netjsonconfig/pkg/ast/vxlan"
	"github.com/honeybbq/netjsonconfig/pkg/netjsonconfig"
	"github.com/honeybbq/netjsonconfig/pkg/nxerrors"
	"github.com/honeybbq/netjsonconfig/pkg/renderer"
)

type Backend struct {
	renderer renderer.Renderer[*ast.Document]
	parser   renderer.Parser[*ast.Document]
}

func New(r renderer.Renderer[*ast.Document], p renderer.Parser[*ast.Document]) *Backend {
	return &Backend{renderer: r, parser: p}
}

func (b *Backend) Name() string {
	return "vxlan"
}

func (b *Backend) ToNative(ctx context.Context, cfg proto.Message, opts netjsonconfig.RenderOptions) (*netjsonconfig.Bundle, error) {
	vxlanCfg, ok := cfg.(*vxlanv1.VxlanConfig)
	if !ok {
		return nil, nxerrors.New(nxerrors.KindValidation, errors.New("expected VxlanConfig payload"))
	}
	domainCfg, err := domain.FromProto(vxlanCfg)
	if err != nil {
		return nil, err
	}
	doc, err := domainCfg.ToAST()
	if err != nil {
		return nil, err
	}
	return b.renderer.Render(ctx, doc, opts)
}

func (b *Backend) ToNetJSON(ctx context.Context, bundle *netjsonconfig.Bundle, opts netjsonconfig.ParseOptions) (proto.Message, error) {
	doc, err := b.parser.Parse(ctx, bundle, opts)
	if err != nil {
		return nil, err
	}
	cfg, err := domain.FromAST(doc)
	if err != nil {
		return nil, err
	}
	return cfg.ToProto()
}
