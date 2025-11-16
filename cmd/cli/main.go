package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	openvpnv1 "github.com/honeybbq/netjson/gen/go/netjson/openvpn/v1"
	openwrtv1 "github.com/honeybbq/netjson/gen/go/netjson/openwrt/v1"
	vxlanv1 "github.com/honeybbq/netjson/gen/go/netjson/vxlan/v1"
	wireguardv1 "github.com/honeybbq/netjson/gen/go/netjson/wireguard/v1"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	openvpnbackend "github.com/honeybbq/netjsonconfig/backend/openvpn"
	openwrtbackend "github.com/honeybbq/netjsonconfig/backend/openwrt"
	vxlanbackend "github.com/honeybbq/netjsonconfig/backend/vxlan"
	wireguardbackend "github.com/honeybbq/netjsonconfig/backend/wireguard"
	"github.com/honeybbq/netjsonconfig/pkg/netjsonconfig"
	openvpnrenderer "github.com/honeybbq/netjsonconfig/pkg/renderer/openvpn"
	ucirenderer "github.com/honeybbq/netjsonconfig/pkg/renderer/uci"
	vxlanrenderer "github.com/honeybbq/netjsonconfig/pkg/renderer/vxlan"
	wireguardrenderer "github.com/honeybbq/netjsonconfig/pkg/renderer/wireguard"
)

type backendEntry struct {
	backend    netjsonconfig.Backend
	newMessage func() proto.Message
}

func main() {
	var (
		mode         = flag.String("mode", "render", "operation mode: render | parse")
		backendName  = flag.String("backend", "", "backend name (openwrt|openvpn|wireguard|vxlan)")
		inputPath    = flag.String("input", "", "input path (default: stdin)")
		outputPath   = flag.String("output", "", "output path (default: stdout)")
		filesOutDir  = flag.String("files-dir", "", "directory for additional files (render mode)")
		prettyJSON   = flag.Bool("pretty", true, "pretty print JSON in parse mode")
		listBackends = flag.Bool("list-backends", false, "list supported backends")
	)
	flag.Parse()

	registry := buildRegistry()
	if *listBackends {
		printBackends(registry)
		return
	}

	if *backendName == "" {
		exitWithError(errors.New("backend is required (use -backend)"))
	}

	entry, ok := registry[strings.ToLower(*backendName)]
	if !ok {
		exitWithError(fmt.Errorf("unknown backend %q", *backendName))
	}

	ctx := context.Background()
	switch strings.ToLower(*mode) {
	case "render":
		payload, err := readInput(*inputPath)
		if err != nil {
			exitWithError(fmt.Errorf("read input: %w", err))
		}
		message := entry.newMessage()
		unmarshal := protojson.UnmarshalOptions{
			DiscardUnknown: false,
		}
		if err := unmarshal.Unmarshal(payload, message); err != nil {
			exitWithError(fmt.Errorf("decode netjson: %w", err))
		}
		bundle, err := entry.backend.ToNative(ctx, message, netjsonconfig.RenderOptions{})
		if err != nil {
			exitWithError(fmt.Errorf("render: %w", err))
		}

		// 处理输出
		if *outputPath == "" || *outputPath == "-" {
			// 输出到 stdout：合并所有包（用于调试）
			if err := writeBundle(os.Stdout, bundle); err != nil {
				exitWithError(fmt.Errorf("write output: %w", err))
			}
		} else {
			// 输出到文件：每个包单独一个文件
			if err := writeBundleToFiles(*outputPath, *filesOutDir, bundle); err != nil {
				exitWithError(err)
			}
		}
	case "parse":
		data, err := readInput(*inputPath)
		if err != nil {
			exitWithError(fmt.Errorf("read input: %w", err))
		}
		bundle := &netjsonconfig.Bundle{
			Packages: []netjsonconfig.Package{{Name: "main", Content: data}},
		}
		msg, err := entry.backend.ToNetJSON(ctx, bundle, netjsonconfig.ParseOptions{})
		if err != nil {
			exitWithError(fmt.Errorf("parse: %w", err))
		}
		marshal := protojson.MarshalOptions{}
		if *prettyJSON {
			marshal.Multiline = true
			marshal.Indent = "  "
		}
		payload, err := marshal.Marshal(msg)
		if err != nil {
			exitWithError(fmt.Errorf("encode json: %w", err))
		}
		if err := writeOutput(*outputPath, payload); err != nil {
			exitWithError(fmt.Errorf("write output: %w", err))
		}
	default:
		exitWithError(fmt.Errorf("unknown mode %q (use render|parse)", *mode))
	}
}

func buildRegistry() map[string]backendEntry {
	return map[string]backendEntry{
		"openwrt": {
			backend: openwrtbackend.New(
				ucirenderer.NewPlainTextRenderer(),
				ucirenderer.NewNotImplementedParser(),
			),
			newMessage: func() proto.Message { return &openwrtv1.OpenWrtConfig{} },
		},
		"openvpn": {
			backend: openvpnbackend.New(
				openvpnrenderer.NewPlainTextRenderer(),
				openvpnrenderer.NewNotImplementedParser(),
			),
			newMessage: func() proto.Message { return &openvpnv1.OpenVpnConfig{} },
		},
		"wireguard": {
			backend: wireguardbackend.New(
				wireguardrenderer.NewPlainTextRenderer(),
				wireguardrenderer.NewNotImplementedParser(),
			),
			newMessage: func() proto.Message { return &wireguardv1.WireguardConfig{} },
		},
		"vxlan": {
			backend: vxlanbackend.New(
				vxlanrenderer.NewPlainTextRenderer(),
				vxlanrenderer.NewNotImplementedParser(),
			),
			newMessage: func() proto.Message { return &vxlanv1.VxlanConfig{} },
		},
	}
}

func printBackends(registry map[string]backendEntry) {
	fmt.Println("Supported backends:")
	for name := range registry {
		fmt.Printf("  - %s\n", name)
	}
}

func readInput(path string) ([]byte, error) {
	if path == "" || path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

func writeOutput(path string, data []byte) error {
	if path == "" || path == "-" {
		_, err := os.Stdout.Write(data)
		if err == nil && (len(data) == 0 || data[len(data)-1] != '\n') {
			_, err = fmt.Fprintln(os.Stdout)
		}
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// writeBundle 将 bundle 输出到 writer（用于 stdout）
func writeBundle(w *os.File, bundle *netjsonconfig.Bundle) error {
	// 输出所有包的内容（用 package 行分隔，便于调试）
	for i, pkg := range bundle.Packages {
		if i > 0 {
			fmt.Fprintln(w)
		}
		if bundle.Metadata.Format == "uci" {
			// UCI 格式输出 package 行（用于调试）
			fmt.Fprintf(w, "package %s\n\n", pkg.Name)
		}
		if _, err := w.Write(pkg.Content); err != nil {
			return err
		}
	}
	return nil
}

// writeBundleToFiles 将 bundle 写入文件系统
func writeBundleToFiles(mainOut, filesDir string, bundle *netjsonconfig.Bundle) error {
	// 根据格式决定如何写入
	if bundle.Metadata.Format == "uci" {
		// UCI 格式：每个包独立文件到 /etc/config/<name>
		baseDir := mainOut
		if baseDir == "" {
			baseDir = "."
		}
		configDir := filepath.Join(baseDir, "etc", "config")

		for _, pkg := range bundle.Packages {
			target := filepath.Join(configDir, pkg.Name)
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return fmt.Errorf("create directories for %q: %w", target, err)
			}
			if err := os.WriteFile(target, pkg.Content, 0o644); err != nil {
				return fmt.Errorf("write package file %q: %w", target, err)
			}
		}
	} else {
		// 其他格式：单文件输出
		if len(bundle.Packages) > 0 {
			if err := os.WriteFile(mainOut, bundle.Packages[0].Content, 0o644); err != nil {
				return fmt.Errorf("write output: %w", err)
			}
		}
	}

	// 写入附加文件
	if len(bundle.Files) > 0 {
		if err := writeBundleFiles(filesDir, bundle.Files); err != nil {
			return err
		}
	}

	return nil
}

// writeBundleFiles 写入附加文件
func writeBundleFiles(dir string, files []netjsonconfig.File) error {
	if dir == "" {
		return fmt.Errorf("additional files produced; specify -files-dir to write them")
	}
	for _, file := range files {
		if file.Path == "" {
			continue
		}
		rel := strings.TrimPrefix(file.Path, "/")
		rel = strings.TrimPrefix(rel, string(filepath.Separator))
		if rel == "" {
			return fmt.Errorf("invalid additional file path %q", file.Path)
		}
		target := filepath.Join(dir, filepath.Clean(rel))
		if !strings.HasPrefix(target, filepath.Clean(dir)) {
			return fmt.Errorf("additional file escapes files-dir: %q", file.Path)
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("create directories for %q: %w", target, err)
		}
		mode := file.Mode
		if mode == 0 {
			mode = 0o644
		}
		if err := os.WriteFile(target, file.Content, mode); err != nil {
			return fmt.Errorf("write additional file %q: %w", target, err)
		}
	}
	return nil
}

func exitWithError(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
