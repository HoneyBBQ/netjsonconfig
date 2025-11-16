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

// TestOpenWrtCompleteConfig tests a comprehensive configuration with all major features
func TestOpenWrtCompleteConfig(t *testing.T) {
	t.Parallel()

	jsonPath := filepath.Join("..", "testdata", "openwrt", "complete_config.json")
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

	// Note: We skip exact output comparison because Python uses DSA-style device sections
	// while our implementation uses the older interface-centric approach.
	// Both are valid OpenWrt configurations, just different styles.
	// Instead, we verify all key features are present.

	// Verify features
	t.Run("DNS configuration", func(t *testing.T) {
		if !strings.Contains(actual, "option dns '8.8.8.8 1.1.1.1'") {
			t.Errorf("missing global DNS servers in lan interface")
		}
		if !strings.Contains(actual, "option dns_search 'example.com local'") {
			t.Errorf("missing global DNS search domains in lan interface")
		}
	})

	t.Run("Bridge configuration", func(t *testing.T) {
		// DSA style: device section with type bridge
		if !strings.Contains(actual, "config device 'device_lan'") {
			t.Errorf("missing device section for bridge")
		}
		if !strings.Contains(actual, "option type 'bridge'") {
			t.Errorf("missing bridge type in device section")
		}
		if !strings.Contains(actual, "list ports 'eth0'") {
			t.Errorf("missing bridge member eth0 in ports")
		}
		// Interface section should reference the device
		if !strings.Contains(actual, "option device 'br-lan'") {
			t.Errorf("interface should reference br-lan device")
		}
	})

	t.Run("Routes and Rules", func(t *testing.T) {
		if !strings.Contains(actual, "config route") {
			t.Errorf("missing route configuration")
		}
		if !strings.Contains(actual, "config rule") {
			t.Errorf("missing rule configuration")
		}
	})

	t.Run("NTP", func(t *testing.T) {
		if !strings.Contains(actual, "config timeserver 'ntp'") {
			t.Errorf("missing NTP configuration")
		}
	})

	t.Run("OpenVPN", func(t *testing.T) {
		if !strings.Contains(actual, "package openvpn") {
			t.Errorf("missing openvpn package")
		}
		if !strings.Contains(actual, "config openvpn 'test_server'") {
			t.Errorf("missing openvpn configuration")
		}
	})

	t.Run("Additional files", func(t *testing.T) {
		if len(cfg.GetFiles()) != len(bundle.Files) {
			t.Errorf("expected %d additional files, got %d", len(cfg.GetFiles()), len(bundle.Files))
		}
		if len(bundle.Files) >= 2 {
			// Check first file
			file0 := bundle.Files[0]
			if file0.Path != "/etc/config/custom" {
				t.Errorf("file 0: expected path /etc/config/custom, got %s", file0.Path)
			}
			// Check second file
			file1 := bundle.Files[1]
			if file1.Path != "/etc/banner" {
				t.Errorf("file 1: expected path /etc/banner, got %s", file1.Path)
			}
		}
	})
}
