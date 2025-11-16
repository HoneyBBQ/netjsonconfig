package openvpn

import (
	"bytes"
	"context"
	"fmt"

	ast "github.com/honeybbq/netjsonconfig/pkg/ast/openvpn"
	"github.com/honeybbq/netjsonconfig/pkg/netjsonconfig"
	"github.com/honeybbq/netjsonconfig/pkg/nxerrors"
)

// PlainTextRenderer 负责将 OpenVPN AST 渲染成文本配置。
type PlainTextRenderer struct{}

func NewPlainTextRenderer() *PlainTextRenderer {
	return &PlainTextRenderer{}
}

func (r *PlainTextRenderer) Render(ctx context.Context, doc *ast.Document, opts netjsonconfig.RenderOptions) (*netjsonconfig.Bundle, error) {
	if doc == nil {
		return nil, nxerrors.New(nxerrors.KindInternal, fmt.Errorf("document is nil"))
	}

	bundle := netjsonconfig.NewBundle("openvpn", "openvpn")

	// OpenVPN 配置作为单个包
	var buf bytes.Buffer
	for idx, inst := range doc.Instances {
		if inst == nil || inst.Name == "" {
			continue
		}
		fmt.Fprintf(&buf, "# openvpn config: %s\n\n", inst.Name)
		for _, dir := range inst.Directives {
			if dir.HasValue {
				fmt.Fprintf(&buf, "%s %s\n", dir.Key, dir.Value)
			} else {
				fmt.Fprintf(&buf, "%s\n", dir.Key)
			}
		}
		if idx < len(doc.Instances)-1 {
			buf.WriteByte('\n')
		}
	}

	if buf.Len() > 0 {
		bundle.Packages = append(bundle.Packages, netjsonconfig.Package{
			Name:    "openvpn",
			Content: buf.Bytes(),
		})
	}

	// 处理附加文件
	if len(doc.Files) > 0 {
		for _, file := range doc.Files {
			if file.Path == "" {
				continue
			}
			bundle.Files = append(bundle.Files, netjsonconfig.File{
				Path:    file.Path,
				Mode:    file.Mode,
				Content: append([]byte(nil), file.Contents...),
			})
		}
	}

	return bundle, nil
}
