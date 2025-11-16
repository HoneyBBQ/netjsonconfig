package vxlan

import (
	"context"
	"fmt"

	ast "github.com/honeybbq/netjsonconfig/pkg/ast/vxlan"
	wireguardast "github.com/honeybbq/netjsonconfig/pkg/ast/wireguard"
	"github.com/honeybbq/netjsonconfig/pkg/netjsonconfig"
	"github.com/honeybbq/netjsonconfig/pkg/nxerrors"
	"github.com/honeybbq/netjsonconfig/pkg/renderer"
	wireguardrenderer "github.com/honeybbq/netjsonconfig/pkg/renderer/wireguard"
)

// PlainTextRenderer 复用 WireGuard 渲染器输出 VXLAN 组合配置。
type PlainTextRenderer struct {
	wireguard renderer.Renderer[*wireguardast.Document]
}

func NewPlainTextRenderer() *PlainTextRenderer {
	return &PlainTextRenderer{
		wireguard: wireguardrenderer.NewPlainTextRenderer(),
	}
}

func (r *PlainTextRenderer) Render(ctx context.Context, doc *ast.Document, opts netjsonconfig.RenderOptions) (*netjsonconfig.Bundle, error) {
	if doc == nil {
		return nil, nxerrors.New(nxerrors.KindInternal, fmt.Errorf("document is nil"))
	}
	if doc.Wireguard == nil {
		return nil, nxerrors.New(nxerrors.KindValidation, fmt.Errorf("wireguard section is required"))
	}
	
	// 复用 WireGuard 渲染器
	bundle, err := r.wireguard.Render(ctx, doc.Wireguard, opts)
	if err != nil {
		return nil, err
	}
	
	// 修改元数据标记为 vxlan
	bundle.Metadata.Format = "vxlan"
	bundle.Metadata.Backend = "vxlan"
	
	return bundle, nil
}
