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

func TestOpenWrtNetworkBridge(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join("..", "testdata", "openwrt", "interface_bridge.json")
	payload, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read netjson: %v", err)
	}

	var device openwrtv1.OpenWrtConfig
	if err := protojson.Unmarshal(payload, &device); err != nil {
		t.Fatalf("unmarshal netjson: %v", err)
	}

	backend := openwrtbackend.New(ucirenderer.NewPlainTextRenderer(), ucirenderer.NewNotImplementedParser())
	bundle, err := backend.ToNative(context.Background(), &device, netjsonconfig.RenderOptions{})
	if err != nil {
		t.Fatalf("ToNative failed: %v", err)
	}

	got := bundleToText(bundle)
	wantBytes, err := os.ReadFile(filepath.Join("..", "testdata", "openwrt", "interface_bridge.uci"))
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}
	want := string(wantBytes)

	if !compareConfigs(got, want) {
		t.Fatalf("%s", formatConfigDiff(got, want))
	}
}

func TestOpenWrtRoutesAndRules(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join("..", "testdata", "openwrt", "routes_rules.json")
	payload, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read netjson: %v", err)
	}

	var device openwrtv1.OpenWrtConfig
	if err := protojson.Unmarshal(payload, &device); err != nil {
		t.Fatalf("unmarshal netjson: %v", err)
	}

	backend := openwrtbackend.New(ucirenderer.NewPlainTextRenderer(), ucirenderer.NewNotImplementedParser())
	bundle, err := backend.ToNative(context.Background(), &device, netjsonconfig.RenderOptions{})
	if err != nil {
		t.Fatalf("ToNative failed: %v", err)
	}

	got := bundleToText(bundle)
	wantBytes, err := os.ReadFile(filepath.Join("..", "testdata", "openwrt", "routes_rules.uci"))
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}
	want := string(wantBytes)

	if !compareConfigs(got, want) {
		t.Fatalf("%s", formatConfigDiff(got, want))
	}
}

func TestOpenWrtWireless(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join("..", "testdata", "openwrt", "wireless.json")
	payload, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read netjson: %v", err)
	}

	var device openwrtv1.OpenWrtConfig
	if err := protojson.Unmarshal(payload, &device); err != nil {
		t.Fatalf("unmarshal netjson: %v", err)
	}

	backend := openwrtbackend.New(ucirenderer.NewPlainTextRenderer(), ucirenderer.NewNotImplementedParser())
	bundle, err := backend.ToNative(context.Background(), &device, netjsonconfig.RenderOptions{})
	if err != nil {
		t.Fatalf("ToNative failed: %v", err)
	}

	got := bundleToText(bundle)
	wantBytes, err := os.ReadFile(filepath.Join("..", "testdata", "openwrt", "wireless.uci"))
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}
	want := string(wantBytes)

	if !compareConfigs(got, want) {
		t.Fatalf("%s", formatConfigDiff(got, want))
	}
}

func TestOpenWrtSystemLeds(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join("..", "testdata", "openwrt", "system_leds.json")
	payload, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read netjson: %v", err)
	}

	var device openwrtv1.OpenWrtConfig
	if err := protojson.Unmarshal(payload, &device); err != nil {
		t.Fatalf("unmarshal netjson: %v", err)
	}

	backend := openwrtbackend.New(ucirenderer.NewPlainTextRenderer(), ucirenderer.NewNotImplementedParser())
	bundle, err := backend.ToNative(context.Background(), &device, netjsonconfig.RenderOptions{})
	if err != nil {
		t.Fatalf("ToNative failed: %v", err)
	}

	got := bundleToText(bundle)
	wantBytes, err := os.ReadFile(filepath.Join("..", "testdata", "openwrt", "system_leds.uci"))
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}
	want := string(wantBytes)

	if !compareConfigs(got, want) {
		t.Fatalf("%s", formatConfigDiff(got, want))
	}
}

func TestOpenWrtSwitches(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join("..", "testdata", "openwrt", "switches.json")
	payload, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read netjson: %v", err)
	}

	var device openwrtv1.OpenWrtConfig
	if err := protojson.Unmarshal(payload, &device); err != nil {
		t.Fatalf("unmarshal netjson: %v", err)
	}

	backend := openwrtbackend.New(ucirenderer.NewPlainTextRenderer(), ucirenderer.NewNotImplementedParser())
	bundle, err := backend.ToNative(context.Background(), &device, netjsonconfig.RenderOptions{})
	if err != nil {
		t.Fatalf("ToNative failed: %v", err)
	}

	got := bundleToText(bundle)
	wantBytes, err := os.ReadFile(filepath.Join("..", "testdata", "openwrt", "switches.uci"))
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}
	want := string(wantBytes)

	if !compareConfigs(got, want) {
		t.Fatalf("%s", formatConfigDiff(got, want))
	}
}

func TestOpenWrtSystemFull(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join("..", "testdata", "openwrt", "system_full.json")
	payload, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read netjson: %v", err)
	}

	var device openwrtv1.OpenWrtConfig
	if err := protojson.Unmarshal(payload, &device); err != nil {
		t.Fatalf("unmarshal netjson: %v", err)
	}

	backend := openwrtbackend.New(ucirenderer.NewPlainTextRenderer(), ucirenderer.NewNotImplementedParser())
	bundle, err := backend.ToNative(context.Background(), &device, netjsonconfig.RenderOptions{})
	if err != nil {
		t.Fatalf("ToNative failed: %v", err)
	}

	got := bundleToText(bundle)
	wantBytes, err := os.ReadFile(filepath.Join("..", "testdata", "openwrt", "system_full.uci"))
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}
	want := string(wantBytes)

	if !compareConfigs(got, want) {
		t.Fatalf("%s", formatConfigDiff(got, want))
	}
}

func TestOpenWrtWireguardInterface(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join("..", "testdata", "openwrt", "wireguard_interface.json")
	payload, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read netjson: %v", err)
	}

	var device openwrtv1.OpenWrtConfig
	if err := protojson.Unmarshal(payload, &device); err != nil {
		t.Fatalf("unmarshal netjson: %v", err)
	}

	backend := openwrtbackend.New(ucirenderer.NewPlainTextRenderer(), ucirenderer.NewNotImplementedParser())
	bundle, err := backend.ToNative(context.Background(), &device, netjsonconfig.RenderOptions{})
	if err != nil {
		t.Fatalf("ToNative failed: %v", err)
	}

	got := bundleToText(bundle)
	wantBytes, err := os.ReadFile(filepath.Join("..", "testdata", "openwrt", "wireguard_interface.uci"))
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}
	want := string(wantBytes)

	if !compareConfigs(got, want) {
		t.Fatalf("%s", formatConfigDiff(got, want))
	}
}

func TestOpenWrtWireguardPeers(t *testing.T) {
	t.Parallel()

	cfgPath := filepath.Join("..", "testdata", "openwrt", "wireguard_peers.json")
	payload, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("read netjson: %v", err)
	}

	var device openwrtv1.OpenWrtConfig
	if err := protojson.Unmarshal(payload, &device); err != nil {
		t.Fatalf("unmarshal netjson: %v", err)
	}

	backend := openwrtbackend.New(ucirenderer.NewPlainTextRenderer(), ucirenderer.NewNotImplementedParser())
	bundle, err := backend.ToNative(context.Background(), &device, netjsonconfig.RenderOptions{})
	if err != nil {
		t.Fatalf("ToNative failed: %v", err)
	}

	got := bundleToText(bundle)
	wantBytes, err := os.ReadFile(filepath.Join("..", "testdata", "openwrt", "wireguard_peers.uci"))
	if err != nil {
		t.Fatalf("read expected: %v", err)
	}
	want := string(wantBytes)

	if !compareConfigs(got, want) {
		t.Fatalf("%s", formatConfigDiff(got, want))
	}
}
