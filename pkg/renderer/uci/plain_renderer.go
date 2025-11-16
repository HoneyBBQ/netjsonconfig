package uci

import (
	"context"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	ast "github.com/honeybbq/netjsonconfig/pkg/ast/uci"
	"github.com/honeybbq/netjsonconfig/pkg/netjsonconfig"
	"github.com/honeybbq/netjsonconfig/pkg/nxerrors"
)

// PlainTextRenderer 将 UCI AST 渲染为纯文本 DSL。
type PlainTextRenderer struct{}

func NewPlainTextRenderer() *PlainTextRenderer {
	return &PlainTextRenderer{}
}

// Render 实现 renderer.Renderer。
func (r *PlainTextRenderer) Render(ctx context.Context, doc *ast.Document, opts netjsonconfig.RenderOptions) (*netjsonconfig.Bundle, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if doc == nil {
		return nil, nxerrors.New(nxerrors.KindRender, fmt.Errorf("uci document is nil"))
	}

	packages := filterPackages(doc.Packages)
	if len(packages) == 0 {
		return nil, nxerrors.New(nxerrors.KindRender, fmt.Errorf("empty document"))
	}

	// 创建 Bundle
	bundle := netjsonconfig.NewBundle("uci", "openwrt")

	// 为每个 UCI 包生成独立的配置内容（不包含 "package" 行）
	for _, pkg := range packages {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		var b strings.Builder
		sections := filterSections(pkg.Sections)
		
		for sectionIndex, section := range sections {
			sectionName := section.Name
			if sectionName == "" {
				sectionName = fmt.Sprintf("%s_%d", section.Type, sectionIndex)
			}
			fmt.Fprintf(&b, "config %s '%s'\n", section.Type, sectionName)

			for _, key := range sortedKeys(section.Options) {
				for _, value := range section.Options[key] {
					fmt.Fprintf(&b, "\toption %s '%s'\n", key, escape(value))
				}
			}
			for _, key := range sortedKeys(section.Lists) {
				for _, value := range section.Lists[key] {
					fmt.Fprintf(&b, "\tlist %s '%s'\n", key, escape(value))
				}
			}

			if sectionIndex < len(sections)-1 {
				b.WriteString("\n")
			}
		}

		content := b.String()
		// 确保以换行符结尾
		if !strings.HasSuffix(content, "\n") {
			content += "\n"
		}

		bundle.Packages = append(bundle.Packages, netjsonconfig.Package{
			Name:    pkg.Name,
			Content: []byte(content),
		})
	}

	// 处理附加文件
	if len(doc.Files) > 0 {
		for _, file := range doc.Files {
			if file == nil {
				continue
			}
			mode, err := parseFileMode(file.GetMode())
			if err != nil {
				return nil, err
			}
			bundle.Files = append(bundle.Files, netjsonconfig.File{
				Path:    file.GetPath(),
				Mode:    mode,
				Content: []byte(file.GetContents()),
			})
		}
	}

	return bundle, nil
}

func parseFileMode(value string) (fs.FileMode, error) {
	if value == "" {
		return 0o644, nil
	}
	parsed, err := strconv.ParseUint(value, 8, 32)
	if err != nil {
		return 0, nxerrors.New(nxerrors.KindValidation, fmt.Errorf("invalid file mode %q: %w", value, err))
	}
	return fs.FileMode(parsed), nil
}

func filterPackages(pkgs []*ast.Package) []*ast.Package {
	filtered := make([]*ast.Package, 0, len(pkgs))
	for _, pkg := range pkgs {
		if pkg == nil || pkg.Name == "" {
			continue
		}
		filtered = append(filtered, pkg)
	}
	return filtered
}

func filterSections(sections []*ast.Section) []*ast.Section {
	filtered := make([]*ast.Section, 0, len(sections))
	for _, sec := range sections {
		if sec == nil || sec.Type == "" {
			continue
		}
		if sec.Options == nil {
			sec.Options = make(map[string][]string)
		}
		if sec.Lists == nil {
			sec.Lists = make(map[string][]string)
		}
		filtered = append(filtered, sec)
	}
	return filtered
}

func sortedKeys(m map[string][]string) []string {
	if len(m) == 0 {
		return nil
	}
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func escape(value string) string {
	return strings.ReplaceAll(value, "'", "\\'")
}
