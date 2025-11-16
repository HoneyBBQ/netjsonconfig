package netjsonconfig

import (
	"encoding/json"
	"testing"
)

func TestDeepMerge_SimpleValues(t *testing.T) {
	base := map[string]any{
		"hostname": "default",
		"timezone": "UTC",
	}
	override := map[string]any{
		"hostname": "Router1",
	}
	
	result := deepMerge(base, override, DefaultIdentifiers)
	
	if result["hostname"] != "Router1" {
		t.Errorf("hostname should be overridden, got %v", result["hostname"])
	}
	if result["timezone"] != "UTC" {
		t.Errorf("timezone should be preserved, got %v", result["timezone"])
	}
}

func TestDeepMerge_NestedDict(t *testing.T) {
	base := map[string]any{
		"general": map[string]any{
			"hostname": "default",
			"timezone": "UTC",
		},
	}
	override := map[string]any{
		"general": map[string]any{
			"hostname": "Router1",
		},
	}
	
	result := deepMerge(base, override, DefaultIdentifiers)
	
	general := result["general"].(map[string]any)
	if general["hostname"] != "Router1" {
		t.Errorf("nested hostname should be overridden")
	}
	if general["timezone"] != "UTC" {
		t.Errorf("timezone should be preserved")
	}
}

func TestDeepMerge_AddNewField(t *testing.T) {
	base := map[string]any{
		"general": map[string]any{"hostname": "Router1"},
	}
	override := map[string]any{
		"wireguard_peers": []any{
			map[string]any{"interface": "wg0"},
		},
	}
	
	result := deepMerge(base, override, DefaultIdentifiers)
	
	if result["general"] == nil {
		t.Error("general should be preserved")
	}
	if result["wireguard_peers"] == nil {
		t.Error("wireguard_peers should be added")
	}
}

func TestMergeSlices_ByName(t *testing.T) {
	base := []any{
		map[string]any{"name": "radio0", "channel": 0, "country": "00"},
	}
	override := []any{
		map[string]any{"name": "radio0", "channel": 10},
	}
	
	result := mergeSlices(base, override, DefaultIdentifiers)
	
	if len(result) != 1 {
		t.Fatalf("should have 1 element, got %d", len(result))
	}
	
	radio := result[0].(map[string]any)
	if radio["name"] != "radio0" {
		t.Error("name mismatch")
	}
	// channel 值应该被覆盖
	if radio["channel"] != 10 {
		t.Errorf("channel should be overridden to 10, got %v (type: %T)", radio["channel"], radio["channel"])
	}
	if radio["country"] != "00" {
		t.Errorf("country should be preserved, got %v", radio["country"])
	}
}

func TestMergeSlices_DifferentNames(t *testing.T) {
	base := []any{
		map[string]any{"name": "wg0", "type": "wireguard"},
	}
	override := []any{
		map[string]any{"name": "lan", "type": "bridge"},
	}
	
	result := mergeSlices(base, override, DefaultIdentifiers)
	
	if len(result) != 2 {
		t.Fatalf("should have 2 elements, got %d", len(result))
	}
}

func TestMergeSlices_SkipDuplicates(t *testing.T) {
	base := []any{
		map[string]any{"mode": "0644", "contents": "test"},
	}
	override := []any{
		map[string]any{"mode": "0644", "contents": "test"},  // 完全相同
		map[string]any{"mode": "0644", "contents": "test2"},
	}
	
	result := mergeSlices(base, override, DefaultIdentifiers)
	
	// 应该跳过重复的第一个，保留第二个
	if len(result) != 2 {
		t.Fatalf("expected 2 elements, got %d", len(result))
	}
}

func TestMergeJSON_Complete(t *testing.T) {
	template1 := []byte(`{
		"dns_servers": ["8.8.8.8"],
		"interfaces": [
			{"name": "wg0", "type": "wireguard", "port": 51820}
		]
	}`)
	
	template2 := []byte(`{
		"ntp": {"enabled": true, "servers": ["pool.ntp.org"]},
		"interfaces": [
			{"name": "lan", "type": "bridge"}
		]
	}`)
	
	config := []byte(`{
		"general": {"hostname": "Router1"},
		"interfaces": [
			{"name": "wg0", "port": 40000}
		]
	}`)
	
	merged, err := MergeJSON([][]byte{template1, template2, config}, DefaultIdentifiers)
	if err != nil {
		t.Fatalf("MergeJSON failed: %v", err)
	}
	
	var result map[string]any
	if err := json.Unmarshal(merged, &result); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	
	// 检查所有字段都存在
	if result["dns_servers"] == nil {
		t.Error("dns_servers should exist from template1")
	}
	if result["ntp"] == nil {
		t.Error("ntp should exist from template2")
	}
	if result["general"] == nil {
		t.Error("general should exist from config")
	}
	
	// 检查 interfaces 合并
	interfaces := result["interfaces"].([]any)
	if len(interfaces) != 2 {
		t.Fatalf("expected 2 interfaces, got %d", len(interfaces))
	}
	
	// 找到 wg0 接口
	var wg0 map[string]any
	for _, iface := range interfaces {
		if m, ok := iface.(map[string]any); ok {
			if m["name"] == "wg0" {
				wg0 = m
				break
			}
		}
	}
	
	if wg0 == nil {
		t.Fatal("wg0 interface not found")
	}
	
	// 检查 wg0 的字段
	if wg0["type"] != "wireguard" {
		t.Errorf("wg0 type should be wireguard from template1")
	}
	if wg0["port"] != float64(40000) {
		t.Errorf("wg0 port should be 40000 from config, got %v", wg0["port"])
	}
}

func TestMergeJSON_MultipleTemplates(t *testing.T) {
	// 模拟 Python 的多层继承
	global := []byte(`{"radios": [{"name": "radio0", "channel": 0, "country": "00"}]}`)
	region := []byte(`{"radios": [{"name": "radio0", "country": "US"}]}`)
	device := []byte(`{"radios": [{"name": "radio0", "channel": 10}]}`)
	
	merged, err := MergeJSON([][]byte{global, region, device}, DefaultIdentifiers)
	if err != nil {
		t.Fatalf("MergeJSON failed: %v", err)
	}
	
	var result map[string]any
	json.Unmarshal(merged, &result)
	
	radios := result["radios"].([]any)
	if len(radios) != 1 {
		t.Fatalf("expected 1 radio, got %d", len(radios))
	}
	
	radio := radios[0].(map[string]any)
	// global 的 channel: 0 被 device 覆盖为 10
	if radio["channel"] != float64(10) {
		t.Errorf("channel should be 10, got %v", radio["channel"])
	}
	// global 的 country: "00" 被 region 覆盖为 "US"
	if radio["country"] != "US" {
		t.Errorf("country should be US, got %v", radio["country"])
	}
}

func TestExtractIdentifier(t *testing.T) {
	tests := []struct {
		name       string
		m          map[string]any
		identifiers []string
		want       any
	}{
		{
			name: "has name",
			m:    map[string]any{"name": "wg0", "type": "wireguard"},
			identifiers: DefaultIdentifiers,
			want: "wg0",
		},
		{
			name: "has config_value",
			m:    map[string]any{"config_value": "test", "type": "something"},
			identifiers: DefaultIdentifiers,
			want: "test",
		},
		{
			name: "has id",
			m:    map[string]any{"id": "123"},
			identifiers: DefaultIdentifiers,
			want: "123",
		},
		{
			name: "no identifier",
			m:    map[string]any{"type": "something"},
			identifiers: DefaultIdentifiers,
			want: nil,
		},
		{
			name: "empty value",
			m:    map[string]any{"name": ""},
			identifiers: DefaultIdentifiers,
			want: nil,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractIdentifier(tt.m, tt.identifiers)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

