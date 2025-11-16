package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	wireguardv1 "github.com/honeybbq/netjson/gen/go/netjson/wireguard/v1"

	"google.golang.org/protobuf/encoding/protojson"

	wireguardbackend "github.com/honeybbq/netjsonconfig/backend/wireguard"
	"github.com/honeybbq/netjsonconfig/pkg/netjsonconfig"
	wireguardrenderer "github.com/honeybbq/netjsonconfig/pkg/renderer/wireguard"
)

func TestWireguardRenderBasic(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join("..", "testdata", "wireguard", "basic.json")
	payload, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	var cfg wireguardv1.WireguardConfig
	if err := protojson.Unmarshal(payload, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	backend := wireguardbackend.New(wireguardrenderer.NewPlainTextRenderer(), wireguardrenderer.NewNotImplementedParser())
	bundle, err := backend.ToNative(context.Background(), &cfg, netjsonconfig.RenderOptions{})
	if err != nil {
		t.Fatalf("ToNative: %v", err)
	}

	got := bundleToText(bundle)
	wantBytes, err := os.ReadFile(filepath.Join("..", "testdata", "wireguard", "basic.conf"))
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	want := string(wantBytes)
	if !compareConfigs(got, want) {
		t.Fatalf("%s", formatConfigDiff(got, want))
	}

	if len(bundle.Files) != len(cfg.GetFiles()) {
		t.Fatalf("expected %d additional files, got %d", len(cfg.GetFiles()), len(bundle.Files))
	}
	if len(bundle.Files) > 0 {
		file := bundle.Files[0]
		if file.Path != "/etc/wireguard/wg.key" {
			t.Fatalf("unexpected file path: %s", file.Path)
		}
		if string(file.Content) != "WGKEY" {
			t.Fatalf("unexpected file contents: %q", string(file.Content))
		}
	}
}
