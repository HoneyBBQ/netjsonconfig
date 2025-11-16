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

// Backend implements NetJSON ↔ UCI conversion for OpenWrt.
// It coordinates between the domain layer (business logic), AST layer (structure),
// and renderer layer (text generation).
type Backend struct {
	renderer renderer.Renderer[*uci.Document]
	parser   renderer.Parser[*uci.Document]
}

// New creates an OpenWrt backend with the given renderer and parser.
// Both renderer and parser must be non-nil.
func New(r renderer.Renderer[*uci.Document], p renderer.Parser[*uci.Document]) *Backend {
	return &Backend{
		renderer: r,
		parser:   p,
	}
}

// Name returns "openwrt" as the backend identifier.
func (b *Backend) Name() string {
	return "openwrt"
}

// ToNative converts NetJSON to UCI configuration.
// Conversion flow: proto.Message → domain.Config → AST → Renderer → Bundle
func (b *Backend) ToNative(ctx context.Context, cfg proto.Message, opts netjsonconfig.RenderOptions) (*netjsonconfig.Bundle, error) {
	owrtCfg, ok := cfg.(*openwrtv1.OpenWrtConfig)
	if !ok {
		return nil, nxerrors.New(nxerrors.KindValidation, errors.New("expected OpenWrtConfig payload"))
	}
	
	// Convert proto to domain model
	domainCfg, err := domain.FromProto(owrtCfg)
	if err != nil {
		return nil, err
	}
	
	// Convert domain model to AST
	doc, err := domainCfg.ToAST()
	if err != nil {
		return nil, err
	}
	
	// Render AST to Bundle
	return b.renderer.Render(ctx, doc, opts)
}

// ToNetJSON converts UCI configuration back to NetJSON.
// Conversion flow: Bundle → Parser → AST → domain.Config → proto.Message
func (b *Backend) ToNetJSON(ctx context.Context, bundle *netjsonconfig.Bundle, opts netjsonconfig.ParseOptions) (proto.Message, error) {
	// Parse Bundle to AST
	doc, err := b.parser.Parse(ctx, bundle, opts)
	if err != nil {
		return nil, err
	}
	
	// Convert AST to domain model
	cfg, err := domain.FromAST(doc)
	if err != nil {
		return nil, err
	}
	
	// Convert domain model to proto
	return cfg.ToProto()
}
