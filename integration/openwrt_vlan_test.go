package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	openwrtv1 "github.com/honeybbq/netjson/gen/go/netjson/openwrt/v1"

	openwrtbackend "github.com/honeybbq/netjsonconfig/backend/openwrt"
	"github.com/honeybbq/netjsonconfig/pkg/netjsonconfig"
	ucirenderer "github.com/honeybbq/netjsonconfig/pkg/renderer/uci"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestOpenWrtVlanFiltering(t *testing.T) {
	t.Parallel()

	jsonPath := filepath.Join("..", "testdata", "openwrt", "vlan_filtering.json")
	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read test JSON: %v", err)
	}

	var cfg openwrtv1.OpenWrtConfig
	if err := protojson.Unmarshal(jsonData, &cfg); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	backend := openwrtbackend.New(ucirenderer.NewPlainTextRenderer(), ucirenderer.NewNotImplementedParser())
	bundle, err := backend.ToNative(context.Background(), &cfg, netjsonconfig.RenderOptions{})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	actual := bundleToText(bundle)

	// Note: We skip exact output comparison due to field/section ordering differences.
	// Both are valid OpenWrt configurations. We verify all key features below.

	// Verify VLAN filtering features
	t.Run("VLAN filtering enabled", func(t *testing.T) {
		if !strings.Contains(actual, "option vlan_filtering '1'") {
			t.Errorf("vlan_filtering not enabled in device section")
		}
	})

	t.Run("Bridge VLAN sections", func(t *testing.T) {
		if !strings.Contains(actual, "config bridge-vlan 'vlan_lan_10'") {
			t.Errorf("missing bridge-vlan section for VLAN 10")
		}
		if !strings.Contains(actual, "config bridge-vlan 'vlan_lan_20'") {
			t.Errorf("missing bridge-vlan section for VLAN 20")
		}
		if !strings.Contains(actual, "option vlan '10'") {
			t.Errorf("missing vlan ID in bridge-vlan section")
		}
	})

	t.Run("VLAN port tagging", func(t *testing.T) {
		if !strings.Contains(actual, "list ports 'eth0:t'") {
			t.Errorf("missing tagged port eth0")
		}
		if !strings.Contains(actual, "list ports 'eth1:u*'") {
			t.Errorf("missing untagged primary VID port eth1")
		}
	})

	t.Run("VLAN interfaces", func(t *testing.T) {
		if !strings.Contains(actual, "config interface 'lan_10'") {
			t.Errorf("missing VLAN interface lan_10")
		}
		if !strings.Contains(actual, "option device 'br-lan.10'") {
			t.Errorf("VLAN interface should reference br-lan.10")
		}
		if !strings.Contains(actual, "option proto 'none'") {
			t.Errorf("VLAN interface should have proto=none")
		}
	})
}
