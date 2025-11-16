package openwrt

import (
	"context"
	"errors"

	openwrtv1 "github.com/honeybbq/netjson/gen/go/netjson/openwrt/v1"

	"google.golang.org/protobuf/proto"

	domain "github.com/honeybbq/netjsonconfig/domain/openwrt"
	"github.com/honeybbq/netjsonconfig/pkg/ast/uci"
	"github.com/honeybbq/netjsonconfig/pkg/netjsonconfig"
	"github.com/honeybbq/netjsonconfig/pkg/nxerrors"
	"github.com/honeybbq/netjsonconfig/pkg/renderer"
)

// Backend 实现 OpenWrt NetJSON ↔ DSL 转换。
type Backend struct {
	renderer renderer.Renderer[*uci.Document]
	parser   renderer.Parser[*uci.Document]
}

// New 构造 Backend。
func New(r renderer.Renderer[*uci.Document], p renderer.Parser[*uci.Document]) *Backend {
	return &Backend{
		renderer: r,
		parser:   p,
	}
}

// Name 实现 Backend 接口。
func (b *Backend) Name() string {
	return "openwrt"
}

// ToNative 实现前向转换。
func (b *Backend) ToNative(ctx context.Context, cfg proto.Message, opts netjsonconfig.RenderOptions) (*netjsonconfig.Bundle, error) {
	owrtCfg, ok := cfg.(*openwrtv1.OpenWrtConfig)
	if !ok {
		return nil, nxerrors.New(nxerrors.KindValidation, errors.New("expected OpenWrtConfig payload"))
	}
	domainCfg, err := domain.FromProto(owrtCfg)
	if err != nil {
		return nil, err
	}
	doc, err := domainCfg.ToAST()
	if err != nil {
		return nil, err
	}
	return b.renderer.Render(ctx, doc, opts)
}

// ToNetJSON 实现反向转换。
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
