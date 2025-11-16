package vxlan

import (
	"fmt"

	commonv1 "github.com/honeybbq/netjson/gen/go/netjson/common/v1"
	vxlanv1 "github.com/honeybbq/netjson/gen/go/netjson/vxlan/v1"
	wireguardv1 "github.com/honeybbq/netjson/gen/go/netjson/wireguard/v1"

	wireguarddomain "github.com/honeybbq/netjsonconfig/domain/wireguard"
	ast "github.com/honeybbq/netjsonconfig/pkg/ast/vxlan"
	wireguardast "github.com/honeybbq/netjsonconfig/pkg/ast/wireguard"
	"github.com/honeybbq/netjsonconfig/pkg/nxerrors"
)

// Config 表示 VXLAN 领域模型。
type Config struct {
	Message *vxlanv1.VxlanConfig
}

func FromProto(msg *vxlanv1.VxlanConfig) (*Config, error) {
	if msg == nil {
		return nil, nxerrors.New(nxerrors.KindValidation, fmt.Errorf("config is nil"))
	}
	return &Config{Message: msg}, nil
}

func (c *Config) ToAST() (*ast.Document, error) {
	if c == nil || c.Message == nil {
		return nil, nxerrors.New(nxerrors.KindValidation, fmt.Errorf("config is nil"))
	}

	doc := &ast.Document{
		Tunnels: buildTunnels(c.Message.GetVxlan()),
	}

	wgDoc, err := buildWireguardDocument(c.Message.GetWireguard(), c.Message.GetFiles())
	if err != nil {
		return nil, err
	}
	doc.Wireguard = wgDoc
	return doc, nil
}

func FromAST(doc *ast.Document) (*Config, error) {
	if doc == nil {
		return nil, nxerrors.New(nxerrors.KindParse, fmt.Errorf("document is nil"))
	}
	return nil, nxerrors.ErrNotImplemented
}

func (c *Config) ToProto() (*vxlanv1.VxlanConfig, error) {
	if c == nil {
		return nil, nxerrors.New(nxerrors.KindInternal, fmt.Errorf("config is nil"))
	}
	return nil, nxerrors.ErrNotImplemented
}

func buildWireguardDocument(tunnels []*wireguardv1.WireguardTunnel, files []*commonv1.IncludedFile) (*wireguardast.Document, error) {
	if len(tunnels) == 0 && len(files) == 0 {
		return nil, nil
	}
	wgMsg := &wireguardv1.WireguardConfig{
		Wireguard: tunnels,
		Files:     files,
	}
	wgCfg, err := wireguarddomain.FromProto(wgMsg)
	if err != nil {
		return nil, err
	}
	return wgCfg.ToAST()
}

func buildTunnels(tunnels []*vxlanv1.VxlanTunnel) []*ast.Tunnel {
	if len(tunnels) == 0 {
		return nil
	}
	result := make([]*ast.Tunnel, 0, len(tunnels))
	for _, t := range tunnels {
		if t == nil || t.GetName() == "" {
			continue
		}
		result = append(result, &ast.Tunnel{
			Name:    t.GetName(),
			AutoVNI: t.GetAutoVni(),
			VNI:     t.GetVni(),
		})
	}
	return result
}
