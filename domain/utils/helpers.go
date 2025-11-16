package common

import (
	"encoding/json"
	"strconv"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	"github.com/honeybbq/netjsonconfig/pkg/ast/uci"
)

// SetString stores a string option if non-empty.
func SetString(section *uci.Section, key, value string) {
	if section == nil || value == "" {
		return
	}
	if section.Options == nil {
		section.Options = make(map[string][]string)
	}
	section.Options[key] = []string{value}
}

// SetStringPtr stores the pointed string if not nil/empty.
func SetStringPtr(section *uci.Section, key string, value *string) {
	if value == nil {
		return
	}
	SetString(section, key, *value)
}

// SetUint32Ptr stores uint32 pointer as decimal string.
func SetUint32Ptr(section *uci.Section, key string, value *uint32) {
	if section == nil || value == nil {
		return
	}
	if section.Options == nil {
		section.Options = make(map[string][]string)
	}
	section.Options[key] = []string{strconv.FormatUint(uint64(*value), 10)}
}

// SetUint32Value stores uint32 value as decimal string if non-zero.
func SetUint32Value(section *uci.Section, key string, value uint32) {
	if section == nil || value == 0 {
		return
	}
	if section.Options == nil {
		section.Options = make(map[string][]string)
	}
	section.Options[key] = []string{strconv.FormatUint(uint64(value), 10)}
}

// SetBool stores bool pointer as "1"/"0".
func SetBool(section *uci.Section, key string, value *bool) {
	if section == nil || value == nil {
		return
	}
	SetBoolValue(section, key, *value)
}

// SetBoolValue stores bool value as "1"/"0".
func SetBoolValue(section *uci.Section, key string, value bool) {
	if section == nil {
		return
	}
	if section.Options == nil {
		section.Options = make(map[string][]string)
	}
	if value {
		section.Options[key] = []string{"1"}
	} else {
		section.Options[key] = []string{"0"}
	}
}

// SetList sets a list option after filtering empty values.
func SetList(section *uci.Section, key string, values []string) {
	if section == nil || len(values) == 0 {
		return
	}
	filtered := make([]string, 0, len(values))
	for _, v := range values {
		if v != "" {
			filtered = append(filtered, v)
		}
	}
	if len(filtered) == 0 {
		return
	}
	if section.Lists == nil {
		section.Lists = make(map[string][]string)
	}
	section.Lists[key] = filtered
}

// AppendList appends a single value to a list option.
func AppendList(section *uci.Section, key string, value string) {
	if section == nil || value == "" {
		return
	}
	if section.Lists == nil {
		section.Lists = make(map[string][]string)
	}
	section.Lists[key] = append(section.Lists[key], value)
}

// OptionExists reports whether option already set.
func OptionExists(section *uci.Section, key string) bool {
	if section == nil {
		return false
	}
	_, ok := section.Options[key]
	return ok
}

// ProtoMessageToMap converts proto message into map via protojson.
func ProtoMessageToMap(msg proto.Message) map[string]any {
	if msg == nil {
		return nil
	}
	marshaler := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}
	data, err := marshaler.Marshal(msg)
	if err != nil {
		return nil
	}
	var values map[string]any
	if err := json.Unmarshal(data, &values); err != nil {
		return nil
	}
	return values
}

// ApplyOptionsFromMap writes entries into section, applying skip rules.
func ApplyOptionsFromMap(section *uci.Section, values map[string]any, skip map[string]struct{}) {
	if len(values) == 0 || section == nil {
		return
	}
	for key, raw := range values {
		if skip != nil {
			if _, ok := skip[key]; ok {
				continue
			}
		}
		switch v := raw.(type) {
		case string:
			SetString(section, key, v)
		case bool:
			SetBoolValue(section, key, v)
		case float64:
			SetString(section, key, strconv.FormatInt(int64(v), 10))
		case []any:
			list := toStringSlice(v)
			if len(list) == 0 {
				continue
			}
			SetList(section, key, list)
		}
	}
}

func toStringSlice(items []any) []string {
	if len(items) == 0 {
		return nil
	}
	result := make([]string, 0, len(items))
	for _, item := range items {
		switch v := item.(type) {
		case string:
			if v != "" {
				result = append(result, v)
			}
		case bool:
			if v {
				result = append(result, "1")
			} else {
				result = append(result, "0")
			}
		case float64:
			result = append(result, strconv.FormatInt(int64(v), 10))
		}
	}
	return result
}
