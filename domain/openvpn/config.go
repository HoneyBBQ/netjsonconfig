package openvpn

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"

	commonv1 "github.com/honeybbq/netjson/gen/go/netjson/common/v1"
	openvpnv1 "github.com/honeybbq/netjson/gen/go/netjson/openvpn/v1"

	"google.golang.org/protobuf/encoding/protojson"

	ast "github.com/honeybbq/netjsonconfig/pkg/ast/openvpn"
	"github.com/honeybbq/netjsonconfig/pkg/nxerrors"
)

// Config 表示 OpenVPN 领域模型。
type Config struct {
	Message *openvpnv1.OpenVpnConfig
}

// FromProto 构造模型。
func FromProto(msg *openvpnv1.OpenVpnConfig) (*Config, error) {
	if msg == nil {
		return nil, nxerrors.New(nxerrors.KindValidation, fmt.Errorf("config is nil"))
	}
	return &Config{Message: msg}, nil
}

// ToAST 转换为 OpenVPN 文档。
func (c *Config) ToAST() (*ast.Document, error) {
	if c == nil || c.Message == nil {
		return nil, nxerrors.New(nxerrors.KindValidation, fmt.Errorf("config is nil"))
	}

	doc := &ast.Document{}
	files, err := convertIncludedFiles(c.Message.GetFiles())
	if err != nil {
		return nil, err
	}
	doc.Files = files

	for _, inst := range c.Message.GetOpenvpn() {
		if inst == nil || inst.GetName() == "" {
			continue
		}
		directives, err := buildOpenvpnDirectives(inst)
		if err != nil {
			return nil, err
		}
		doc.Instances = append(doc.Instances, &ast.Instance{
			Name:       inst.GetName(),
			Directives: directives,
		})
	}
	return doc, nil
}

// FromAST 解析 AST（暂未实现）。
func FromAST(doc *ast.Document) (*Config, error) {
	if doc == nil {
		return nil, nxerrors.New(nxerrors.KindParse, fmt.Errorf("document is nil"))
	}
	return nil, nxerrors.ErrNotImplemented
}

// ToProto 输出 NetJSON（暂未实现）。
func (c *Config) ToProto() (*openvpnv1.OpenVpnConfig, error) {
	if c == nil {
		return nil, nxerrors.New(nxerrors.KindInternal, fmt.Errorf("config is nil"))
	}
	return nil, nxerrors.ErrNotImplemented
}

type directiveValue struct {
	value    string
	hasValue bool
}

func buildOpenvpnDirectives(inst *openvpnv1.OpenVpnInstance) ([]ast.Directive, error) {
	marshaller := protojson.MarshalOptions{
		UseProtoNames:   true,
		EmitUnpopulated: false,
	}
	payload, err := marshaller.Marshal(inst)
	if err != nil {
		return nil, nxerrors.New(nxerrors.KindInternal, fmt.Errorf("marshal instance: %w", err))
	}

	raw := make(map[string]any)
	if err := json.Unmarshal(payload, &raw); err != nil {
		return nil, nxerrors.New(nxerrors.KindInternal, fmt.Errorf("decode instance: %w", err))
	}

	delete(raw, "name")

	if err := normalizeRemote(raw); err != nil {
		return nil, err
	}
	normalizeDataCiphers(raw)

	if !hasNonEmptyValue(raw["status"]) {
		delete(raw, "status_version")
	}

	values := make(map[string][]directiveValue)
	keys := make([]string, 0, len(raw))
	for key := range raw {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	zeroAllowed := map[string]struct{}{
		"script_security": {},
	}
	emptyFlagKeys := map[string]struct{}{
		"server_bridge": {},
	}

	for _, key := range keys {
		value := raw[key]
		switch typed := value.(type) {
		case string:
			if strings.TrimSpace(typed) == "" {
				if _, ok := emptyFlagKeys[key]; ok {
					appendDirective(values, key, "", false)
				}
				continue
			}
			appendDirective(values, key, strings.TrimSpace(typed), true)
		case float64:
			if typed == 0 {
				if _, ok := zeroAllowed[key]; !ok {
					continue
				}
			}
			appendDirective(values, key, formatNumber(typed), true)
		case bool:
			if typed {
				appendDirective(values, key, "", false)
			}
		case []string:
			for _, item := range typed {
				item = strings.TrimSpace(item)
				if item == "" {
					continue
				}
				appendDirective(values, key, item, true)
			}
		default:
			// unsupported types (nested objects) are ignored
		}
	}

	var directives []ast.Directive
	for _, key := range keys {
		entries, ok := values[key]
		if !ok {
			continue
		}
		normalized := normalizeKey(key)
		for _, entry := range entries {
			directives = append(directives, ast.Directive{
				Key:      normalized,
				Value:    entry.value,
				HasValue: entry.hasValue,
			})
		}
	}
	return directives, nil
}

func appendDirective(values map[string][]directiveValue, key, value string, hasValue bool) {
	values[key] = append(values[key], directiveValue{
		value:    value,
		hasValue: hasValue,
	})
}

func normalizeRemote(raw map[string]any) error {
	remoteRaw, ok := raw["remote"]
	if !ok {
		return nil
	}
	items, ok := remoteRaw.([]any)
	if !ok {
		delete(raw, "remote")
		return nil
	}
	var entries []string
	for _, item := range items {
		obj, _ := item.(map[string]any)
		if obj == nil {
			continue
		}
		host := strings.TrimSpace(asString(obj["host"]))
		if host == "" {
			continue
		}
		port := formatNumberInterface(obj["port"])
		line := host
		if port != "" {
			line = strings.TrimSpace(line + " " + port)
		}
		if protoVal := strings.TrimSpace(asString(obj["proto"])); protoVal != "" && protoVal != "auto" {
			line = strings.TrimSpace(line + " " + protoVal)
		}
		entries = append(entries, line)
	}
	if len(entries) == 0 {
		delete(raw, "remote")
		return nil
	}
	raw["remote"] = entries
	return nil
}

func normalizeDataCiphers(raw map[string]any) {
	value, ok := raw["data_ciphers"]
	if !ok {
		return
	}
	items, ok := value.([]any)
	if !ok {
		delete(raw, "data_ciphers")
		return
	}
	var ciphers []string
	for _, item := range items {
		obj, _ := item.(map[string]any)
		if obj == nil {
			continue
		}
		cipher := strings.TrimSpace(asString(obj["cipher"]))
		if cipher == "" {
			continue
		}
		if optional, _ := obj["optional"].(bool); optional {
			cipher = "?" + cipher
		}
		ciphers = append(ciphers, cipher)
	}
	if len(ciphers) == 0 {
		delete(raw, "data_ciphers")
		return
	}
	raw["data_ciphers"] = strings.Join(ciphers, ":")
}

func asString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return formatNumber(v)
	default:
		return ""
	}
}

func formatNumberInterface(value any) string {
	if value == nil {
		return ""
	}
	if v, ok := value.(float64); ok {
		if v == 0 {
			return ""
		}
		return formatNumber(v)
	}
	return asString(value)
}

func formatNumber(v float64) string {
	return strconv.FormatInt(int64(v), 10)
}

func normalizeKey(key string) string {
	return strings.ReplaceAll(key, "_", "-")
}

func hasNonEmptyValue(value any) bool {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) != ""
	default:
		return v != nil
	}
}

func convertIncludedFiles(files []*commonv1.IncludedFile) ([]ast.File, error) {
	if len(files) == 0 {
		return nil, nil
	}
	result := make([]ast.File, 0, len(files))
	for _, file := range files {
		if file == nil || file.GetPath() == "" {
			continue
		}
		mode, err := parseFileMode(file.GetMode())
		if err != nil {
			return nil, err
		}
		result = append(result, ast.File{
			Path:     file.GetPath(),
			Mode:     mode,
			Contents: []byte(file.GetContents()),
		})
	}
	return result, nil
}

func parseFileMode(value string) (fs.FileMode, error) {
	if value == "" {
		return 0o644, nil
	}
	parsed, err := strconv.ParseUint(value, 8, 32)
	if err != nil {
		return 0, nxerrors.New(nxerrors.KindValidation, fmt.Errorf("invalid file mode %q: %w", value, err))
	}
	return fs.FileMode(parsed), nil
}
