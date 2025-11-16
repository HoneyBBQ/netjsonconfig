package wireguard

import (
	"bytes"
	"context"
	"fmt"
	"sort"

	ast "github.com/honeybbq/netjsonconfig/pkg/ast/wireguard"
	"github.com/honeybbq/netjsonconfig/pkg/netjsonconfig"
	"github.com/honeybbq/netjsonconfig/pkg/nxerrors"
)

// PlainTextRenderer 渲染 WireGuard AST。
type PlainTextRenderer struct{}

func NewPlainTextRenderer() *PlainTextRenderer {
	return &PlainTextRenderer{}
}

func (r *PlainTextRenderer) Render(ctx context.Context, doc *ast.Document, opts netjsonconfig.RenderOptions) (*netjsonconfig.Bundle, error) {
	if doc == nil {
		return nil, nxerrors.New(nxerrors.KindInternal, fmt.Errorf("document is nil"))
	}

	bundle := netjsonconfig.NewBundle("wireguard", "wireguard")

	// WireGuard 配置作为单个包
	var buf bytes.Buffer
	for _, iface := range doc.Interfaces {
		if iface == nil || iface.Name == "" {
			continue
		}
		fmt.Fprintf(&buf, "# wireguard config: %s\n\n", iface.Name)
		buf.WriteString("[Interface]\n")
		writeDirectiveBlock(&buf, iface.Directives)
		buf.WriteByte('\n')

		for _, peer := range iface.Peers {
			if peer == nil {
				continue
			}
			buf.WriteString("[Peer]\n")
			writeDirectiveBlock(&buf, peer.Directives)
			buf.WriteByte('\n')
		}
	}

	if buf.Len() > 0 {
		bundle.Packages = append(bundle.Packages, netjsonconfig.Package{
			Name:    "wireguard",
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

func writeDirectiveBlock(buf *bytes.Buffer, directives map[string]string) {
	if len(directives) == 0 {
		return
	}
	keys := make([]string, 0, len(directives))
	for k := range directives {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		value := directives[key]
		fmt.Fprintf(buf, "%s = %s\n", key, value)
	}
}
