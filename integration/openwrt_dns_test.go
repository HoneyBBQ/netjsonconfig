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

func TestOpenWrtDNSAndOpenvpn(t *testing.T) {
	t.Parallel()

	// Load test JSON
	jsonPath := filepath.Join("..", "testdata", "openwrt", "dns_openvpn.json")
	jsonData, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("failed to read test JSON: %v", err)
	}

	// Parse into proto
	var cfg openwrtv1.OpenWrtConfig
	if err := protojson.Unmarshal(jsonData, &cfg); err != nil {
		t.Fatalf("failed to unmarshal JSON: %v", err)
	}

	// Render
	backend := openwrtbackend.New(ucirenderer.NewPlainTextRenderer(), ucirenderer.NewNotImplementedParser())
	bundle, err := backend.ToNative(context.Background(), &cfg, netjsonconfig.RenderOptions{})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	// Load expected output
	expectedPath := filepath.Join("..", "testdata", "openwrt", "dns_openvpn.uci")
	expectedData, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read expected output: %v", err)
	}
	expected := string(expectedData)
	actual := bundleToText(bundle)

	// Compare
	if !compareUCIOutput(expected, actual) {
		t.Errorf("output mismatch\nExpected:\n%s\n\nActual:\n%s", expected, actual)
	}

	// Check additional files
	if len(cfg.GetFiles()) != len(bundle.Files) {
		t.Errorf("expected %d additional files, got %d", len(cfg.GetFiles()), len(bundle.Files))
	}

	if len(bundle.Files) > 0 {
		file := bundle.Files[0]
		if file.Path != "/etc/config/custom" {
			t.Errorf("expected file path /etc/config/custom, got %s", file.Path)
		}
		if string(file.Content) != "test content\n" {
			t.Errorf("expected file contents 'test content\\n', got %q", string(file.Content))
		}
	}
}

// compareUCIOutput normalizes and compares UCI output
func compareUCIOutput(expected, actual string) bool {
	return normalizeUCI(expected) == normalizeUCI(actual)
}

// normalizeUCI removes leading/trailing whitespace and normalizes line endings
func normalizeUCI(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "\r\n", "\n")
	// Normalize multiple blank lines to single
	lines := strings.Split(s, "\n")
	var normalized []string
	prevBlank := false
	for _, line := range lines {
		isBlank := strings.TrimSpace(line) == ""
		if isBlank && prevBlank {
			continue
		}
		normalized = append(normalized, line)
		prevBlank = isBlank
	}
	return strings.Join(normalized, "\n")
}
