package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	openvpnv1 "github.com/honeybbq/netjson/gen/go/netjson/openvpn/v1"

	"google.golang.org/protobuf/encoding/protojson"

	openvpnbackend "github.com/honeybbq/netjsonconfig/backend/openvpn"
	"github.com/honeybbq/netjsonconfig/pkg/netjsonconfig"
	openvpnrenderer "github.com/honeybbq/netjsonconfig/pkg/renderer/openvpn"
)

func TestOpenVpnRenderServer(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join("..", "testdata", "openvpn", "server.json")
	payload, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}

	var cfg openvpnv1.OpenVpnConfig
	if err := protojson.Unmarshal(payload, &cfg); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	backend := openvpnbackend.New(openvpnrenderer.NewPlainTextRenderer(), openvpnrenderer.NewNotImplementedParser())
	bundle, err := backend.ToNative(context.Background(), &cfg, netjsonconfig.RenderOptions{})
	if err != nil {
		t.Fatalf("ToNative: %v", err)
	}

	got := bundleToText(bundle)
	wantBytes, err := os.ReadFile(filepath.Join("..", "testdata", "openvpn", "server.conf"))
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
		if file.Path != "/etc/openvpn/ca.pem" {
			t.Fatalf("unexpected file path: %s", file.Path)
		}
		if string(file.Content) != "-----BEGIN-----" {
			t.Fatalf("unexpected file contents: %q", string(file.Content))
		}
	}
}
