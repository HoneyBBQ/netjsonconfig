package integration

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/protojson"

	openwrtv1 "github.com/honeybbq/netjson/gen/go/netjson/openwrt/v1"

	openwrtbackend "github.com/honeybbq/netjsonconfig/backend/openwrt"
	"github.com/honeybbq/netjsonconfig/pkg/netjsonconfig"
	ucirenderer "github.com/honeybbq/netjsonconfig/pkg/renderer/uci"
)

func TestMergeConfigs_WireguardTemplate(t *testing.T) {
	t.Parallel()

	// 创建临时配置文件
	tmpDir := t.TempDir()

	// 基础配置：只有主机名
	baseConfig := `{
		"general": {
			"hostname": "Router1",
			"timezone": "UTC"
		}
	}`
	baseFile := filepath.Join(tmpDir, "base.json")
	if err := os.WriteFile(baseFile, []byte(baseConfig), 0644); err != nil {
		t.Fatal(err)
	}

	// WireGuard 模板
	wgTemplate := `{
		"interfaces": [
			{
				"name": "wg0",
				"type": "wireguard",
				"wireguard": {
					"private_key": "test-key",
					"listen_port": 51820
				},
				"addresses": [
					{
						"proto": "static",
						"family": "ipv4",
						"address": "10.0.0.1",
						"mask": 24
					}
				]
			}
		]
	}`
	wgFile := filepath.Join(tmpDir, "wireguard.json")
	if err := os.WriteFile(wgFile, []byte(wgTemplate), 0644); err != nil {
		t.Fatal(err)
	}

	// 合并配置
	configs := [][]byte{
		[]byte(baseConfig),
		[]byte(wgTemplate),
	}

	merged, err := netjsonconfig.MergeJSON(configs, netjsonconfig.DefaultIdentifiers)
	if err != nil {
		t.Fatalf("MergeJSON failed: %v", err)
	}

	// 解析为 Proto
	var msg openwrtv1.OpenWrtConfig
	if err := protojson.Unmarshal(merged, &msg); err != nil {
		t.Fatalf("unmarshal proto: %v", err)
	}

	// 验证合并结果
	if msg.GetGeneral().GetHostname() != "Router1" {
		t.Errorf("hostname mismatch: got %s", msg.GetGeneral().GetHostname())
	}
	if msg.GetGeneral().GetTimezone() != "UTC" {
		t.Errorf("timezone mismatch: got %s", msg.GetGeneral().GetTimezone())
	}
	if len(msg.GetInterfaces()) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(msg.GetInterfaces()))
	}
	if msg.GetInterfaces()[0].GetName() != "wg0" {
		t.Errorf("interface name mismatch")
	}
	if msg.GetInterfaces()[0].GetType() != "wireguard" {
		t.Errorf("interface type mismatch")
	}
}

func TestMergeConfigs_OverrideInterface(t *testing.T) {
	t.Parallel()

	// 模板：WireGuard 默认端口
	template := []byte(`{
		"interfaces": [
			{
				"name": "wg0",
				"type": "wireguard",
				"wireguard": {
					"private_key": "default-key",
					"listen_port": 51820
				}
			}
		]
	}`)

	// 配置：覆盖端口和密钥
	config := []byte(`{
		"interfaces": [
			{
				"name": "wg0",
				"wireguard": {
					"listen_port": 40000,
					"private_key": "new-key"
				}
			}
		]
	}`)

	merged, err := netjsonconfig.MergeJSON([][]byte{template, config}, netjsonconfig.DefaultIdentifiers)
	if err != nil {
		t.Fatalf("MergeJSON failed: %v", err)
	}

	var msg openwrtv1.OpenWrtConfig
	if err := protojson.Unmarshal(merged, &msg); err != nil {
		t.Fatalf("unmarshal proto: %v", err)
	}

	if len(msg.GetInterfaces()) != 1 {
		t.Fatalf("expected 1 interface, got %d", len(msg.GetInterfaces()))
	}

	iface := msg.GetInterfaces()[0]
	if iface.GetName() != "wg0" {
		t.Errorf("name mismatch")
	}
	if iface.GetType() != "wireguard" {
		t.Errorf("type should be preserved from template")
	}

	wg := iface.GetWireguard()
	if wg == nil {
		t.Fatal("wireguard config is nil")
	}
	if wg.GetListenPort() != 40000 {
		t.Errorf("port should be overridden to 40000, got %d", wg.GetListenPort())
	}
	if wg.GetPrivateKey() != "new-key" {
		t.Errorf("private_key should be overridden to 'new-key', got %s", wg.GetPrivateKey())
	}
}

func TestMergeConfigs_Render(t *testing.T) {
	t.Parallel()

	// 多层配置
	global := []byte(`{
		"dns_servers": ["8.8.8.8", "1.1.1.1"],
		"ntp": {
			"enabled": true,
			"servers": ["pool.ntp.org"]
		}
	}`)

	device := []byte(`{
		"general": {"hostname": "TestRouter"},
		"interfaces": [
			{
				"name": "lan",
				"type": "bridge",
				"bridge_members": ["eth0"],
				"proto": "static",
				"addresses": [
					{
						"family": "ipv4",
						"address": "192.168.1.1",
						"mask": 24
					}
				]
			}
		]
	}`)

	merged, err := netjsonconfig.MergeJSON([][]byte{global, device}, netjsonconfig.DefaultIdentifiers)
	if err != nil {
		t.Fatalf("MergeJSON failed: %v", err)
	}

	var msg openwrtv1.OpenWrtConfig
	if err := protojson.Unmarshal(merged, &msg); err != nil {
		t.Fatalf("unmarshal proto: %v", err)
	}

	// 渲染
	backend := openwrtbackend.New(
		ucirenderer.NewPlainTextRenderer(),
		ucirenderer.NewNotImplementedParser(),
	)
	bundle, err := backend.ToNative(context.Background(), &msg, netjsonconfig.RenderOptions{})
	if err != nil {
		t.Fatalf("ToNative failed: %v", err)
	}

	// 验证生成的配置
	text := bundleToText(bundle)

	// 应该包含 hostname
	if !strings.Contains(text, "hostname 'TestRouter'") {
		t.Error("missing hostname from device config")
	}

	// 应该包含 DNS
	if !strings.Contains(text, "dns '8.8.8.8 1.1.1.1'") {
		t.Error("missing dns_servers from global config")
	}

	// 应该包含 NTP
	if !strings.Contains(text, "list servers 'pool.ntp.org'") {
		t.Error("missing ntp from global config")
	}

	// 应该包含 lan 接口
	if !strings.Contains(text, "config interface 'lan'") {
		t.Error("missing lan interface")
	}
}
