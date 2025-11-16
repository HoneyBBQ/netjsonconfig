package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	vxlanv1 "github.com/honeybbq/netjson/gen/go/netjson/vxlan/v1"

	"google.golang.org/protobuf/encoding/protojson"

	vxlanbackend "github.com/honeybbq/netjsonconfig/backend/vxlan"
	"github.com/honeybbq/netjsonconfig/pkg/netjsonconfig"
	vxlanrenderer "github.com/honeybbq/netjsonconfig/pkg/renderer/vxlan"
)

func TestVxlanWireguardRender(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join("..", "testdata", "vxlan", "basic.json")
	payload, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	var cfg vxlanv1.VxlanConfig
	if err := protojson.Unmarshal(payload, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	backend := vxlanbackend.New(vxlanrenderer.NewPlainTextRenderer(), vxlanrenderer.NewNotImplementedParser())
	bundle, err := backend.ToNative(context.Background(), &cfg, netjsonconfig.RenderOptions{})
	if err != nil {
		t.Fatalf("ToNative: %v", err)
	}

	got := bundleToText(bundle)
	wantBytes, err := os.ReadFile(filepath.Join("..", "testdata", "vxlan", "basic.conf"))
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
		if file.Path != "/etc/vxlan/wg.key" {
			t.Fatalf("unexpected file path: %s", file.Path)
		}
		if string(file.Content) != "WGKEY-VXLAN" {
			t.Fatalf("unexpected file contents: %q", string(file.Content))
		}
	}
}
