package netjsonconfig

import (
	"encoding/json"
	"fmt"
)

// DefaultIdentifiers defines the field names used to match array elements during merge.
// When merging arrays of objects, elements are considered "the same" if they have
// matching values for any of these fields (checked in order).
// This follows the same convention as Python netjsonconfig.
var DefaultIdentifiers = []string{"name", "config_value", "id"}

// MergeJSON merges multiple JSON configurations with later configs overriding earlier ones.
// All inputs must be valid JSON byte arrays representing configuration objects.
//
// Merge rules:
//   - Simple values (string, number, bool): later value overwrites earlier
//   - Objects (maps): recursively merged, with later keys overriding earlier
//   - Arrays: intelligently merged using identifier matching (see DefaultIdentifiers)
//
// Returns the merged configuration as JSON bytes, or an error if any config is invalid.
func MergeJSON(configs [][]byte, identifiers []string) ([]byte, error) {
	if len(configs) == 0 {
		return nil, fmt.Errorf("no configs to merge")
	}
	
	if identifiers == nil {
		identifiers = DefaultIdentifiers
	}
	
	// Parse all configs into maps
	var maps []map[string]any
	for i, cfg := range configs {
		var m map[string]any
		if err := json.Unmarshal(cfg, &m); err != nil {
			return nil, fmt.Errorf("unmarshal config[%d]: %w", i, err)
		}
		maps = append(maps, m)
	}
	
	// Merge layer by layer
	result := make(map[string]any)
	for _, m := range maps {
		result = deepMerge(result, m, identifiers)
	}
	
	// Serialize back to JSON
	return json.Marshal(result)
}

// deepMerge performs a deep merge of two maps, with override taking precedence over base.
//
// Merge strategy by type:
//   - Simple values: override replaces base
//   - Maps (objects): recursively merged
//   - Slices (arrays): intelligently merged using identifier matching
//
// This function is the core of the configuration template system,
// enabling multi-layer configuration inheritance (base → regional → device-specific).
func deepMerge(base, override map[string]any, identifiers []string) map[string]any {
	if base == nil {
		return deepCopy(override)
	}
	if override == nil {
		return deepCopy(base)
	}
	
	result := deepCopy(base)
	
	for key, overrideVal := range override {
		baseVal, exists := result[key]
		
		if !exists {
			// New field, add directly
			result[key] = deepCopyValue(overrideVal)
			continue
		}
		
		// Merge strategy based on type
		switch overrideVal := overrideVal.(type) {
		case map[string]any:
			// Maps: recursive merge
			if baseMap, ok := baseVal.(map[string]any); ok {
				result[key] = deepMerge(baseMap, overrideVal, identifiers)
			} else {
				result[key] = deepCopyValue(overrideVal)
			}
			
		case []any:
			// Arrays: intelligent merge using identifiers
			if baseSlice, ok := baseVal.([]any); ok {
				result[key] = mergeSlices(baseSlice, overrideVal, identifiers)
			} else {
				result[key] = deepCopyValue(overrideVal)
			}
			
		default:
			// Simple values: override replaces base
			result[key] = deepCopyValue(overrideVal)
		}
	}
	
	return result
}

// mergeSlices intelligently merges two slices.
//
// If slice elements are maps containing any of the identifier fields,
// elements with matching identifier values are merged together.
// Elements without matching identifiers are appended.
// Exact duplicates are skipped to avoid redundant entries.
//
// Example with identifiers=["name"]:
//
//	base:     [{"name": "wg0", "port": 51820}]
//	override: [{"name": "wg0", "port": 40000}, {"name": "lan", ...}]
//	result:   [{"name": "wg0", "port": 40000}, {"name": "lan", ...}]
func mergeSlices(base, override []any, identifiers []string) []any {
	if len(base) == 0 {
		return deepCopySlice(override)
	}
	if len(override) == 0 {
		return deepCopySlice(base)
	}
	
	// 建立 base 数组的索引（按标识符）
	baseIndex := make(map[any]int)
	for i, el := range base {
		if m, ok := el.(map[string]any); ok {
			if id := extractIdentifier(m, identifiers); id != nil {
				baseIndex[id] = i
			}
		}
	}
	
	// 复制 base 数组
	result := deepCopySlice(base)
	
	// 处理 override 数组
	for _, overrideEl := range override {
		// 检查是否是重复元素（完全相同）
		if isDuplicate(result, overrideEl) {
			continue
		}
		
		// 如果是字典，尝试按标识符匹配
		if m, ok := overrideEl.(map[string]any); ok {
			id := extractIdentifier(m, identifiers)
			if id != nil {
				if idx, found := baseIndex[id]; found {
					// 找到匹配元素，合并
					if baseMap, ok := result[idx].(map[string]any); ok {
						result[idx] = deepMerge(baseMap, m, identifiers)
						continue
					}
				}
			}
		}
		
		// 没有匹配，追加到结果
		result = append(result, deepCopyValue(overrideEl))
	}
	
	return result
}

// extractIdentifier 从 map 中提取标识符的值。
// 按 identifiers 的顺序查找，返回第一个找到的值。
func extractIdentifier(m map[string]any, identifiers []string) any {
	for _, key := range identifiers {
		if val, ok := m[key]; ok && val != nil && val != "" {
			return val
		}
	}
	return nil
}

// isDuplicate checks if an element already exists in a slice (exact match).
// Uses JSON serialization for deep equality comparison.
func isDuplicate(slice []any, el any) bool {
	elJSON, err := json.Marshal(el)
	if err != nil {
		return false
	}
	
	for _, item := range slice {
		itemJSON, err := json.Marshal(item)
		if err != nil {
			continue
		}
		if string(elJSON) == string(itemJSON) {
			return true
		}
	}
	return false
}

// deepCopy creates a deep copy of a map.
func deepCopy(m map[string]any) map[string]any {
	if m == nil {
		return nil
	}
	result := make(map[string]any, len(m))
	for k, v := range m {
		result[k] = deepCopyValue(v)
	}
	return result
}

// deepCopySlice creates a deep copy of a slice.
func deepCopySlice(s []any) []any {
	if s == nil {
		return nil
	}
	result := make([]any, len(s))
	for i, v := range s {
		result[i] = deepCopyValue(v)
	}
	return result
}

// deepCopyValue creates a deep copy of any value.
// Recursively handles maps and slices; simple types (int, string, bool) are returned as-is
// since they are value types in Go.
func deepCopyValue(v any) any {
	if v == nil {
		return nil
	}
	
	switch val := v.(type) {
	case map[string]any:
		return deepCopy(val)
	case []any:
		return deepCopySlice(val)
	default:
		// Simple types are value types, safe to return directly
		return val
	}
}

