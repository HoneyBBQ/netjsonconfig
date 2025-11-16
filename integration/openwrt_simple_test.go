package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	openwrtv1 "github.com/honeybbq/netjson/gen/go/netjson/openwrt/v1"

	openwrtbackend "github.com/honeybbq/netjsonconfig/backend/openwrt"
	"github.com/honeybbq/netjsonconfig/pkg/netjsonconfig"
	ucirenderer "github.com/honeybbq/netjsonconfig/pkg/renderer/uci"
)

func TestOpenWrtGeneralHostname(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join("..", "testdata", "openwrt", "system_simple.json")
	raw, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read netjson: %v", err)
	}

	var device openwrtv1.OpenWrtConfig
	if err := protojson.Unmarshal(raw, &device); err != nil {
		t.Fatalf("unmarshal netjson: %v", err)
	}

	backend := openwrtbackend.New(ucirenderer.NewPlainTextRenderer(), ucirenderer.NewNotImplementedParser())
	bundle, err := backend.ToNative(context.Background(), &device, netjsonconfig.RenderOptions{})
	if err != nil {
		t.Fatalf("ToNative failed: %v", err)
	}

	got := bundleToText(bundle)
	wantPath := filepath.Join("..", "testdata", "openwrt", "system_simple.uci")
	wantBytes, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatalf("read expected uci: %v", err)
	}
	want := string(wantBytes)

	if !compareConfigs(got, want) {
		t.Fatalf("%s", formatConfigDiff(got, want))
	}
}
