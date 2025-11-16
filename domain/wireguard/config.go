package wireguard

import (
	"fmt"
	"io/fs"
	"strconv"
	"strings"

	commonv1 "github.com/honeybbq/netjson/gen/go/netjson/common/v1"
	wireguardv1 "github.com/honeybbq/netjson/gen/go/netjson/wireguard/v1"

	ast "github.com/honeybbq/netjsonconfig/pkg/ast/wireguard"
	"github.com/honeybbq/netjsonconfig/pkg/nxerrors"
)

// Config 表示 WireGuard 领域模型。
type Config struct {
	Message *wireguardv1.WireguardConfig
}

func FromProto(msg *wireguardv1.WireguardConfig) (*Config, error) {
	if msg == nil {
		return nil, nxerrors.New(nxerrors.KindValidation, fmt.Errorf("config is nil"))
	}
	return &Config{Message: msg}, nil
}

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

	for _, tunnel := range c.Message.GetWireguard() {
		if tunnel == nil || tunnel.GetName() == "" {
			continue
		}
		doc.Interfaces = append(doc.Interfaces, buildInterface(tunnel))
	}
	return doc, nil
}

func FromAST(doc *ast.Document) (*Config, error) {
	if doc == nil {
		return nil, nxerrors.New(nxerrors.KindParse, fmt.Errorf("document is nil"))
	}
	return nil, nxerrors.ErrNotImplemented
}

func (c *Config) ToProto() (*wireguardv1.WireguardConfig, error) {
	if c == nil {
		return nil, nxerrors.New(nxerrors.KindInternal, fmt.Errorf("config is nil"))
	}
	return nil, nxerrors.ErrNotImplemented
}

func buildInterface(tunnel *wireguardv1.WireguardTunnel) *ast.Interface {
	directives := make(map[string]string)
	setString(directives, "Address", tunnel.GetAddress())
	if port := tunnel.GetPort(); port != 0 {
		directives["ListenPort"] = strconv.FormatUint(uint64(port), 10)
	}
	if key := tunnel.GetPrivateKey(); key != "" {
		directives["PrivateKey"] = key
	}
	if dns := tunnel.GetDns(); len(dns) > 0 {
		directives["DNS"] = strings.Join(dns, ",")
	}
	if mtu := tunnel.GetMtu(); mtu != 0 {
		directives["MTU"] = strconv.FormatUint(uint64(mtu), 10)
	}
	if table := tunnel.GetTable(); table != "" {
		directives["Table"] = table
	}
	if tunnel.SaveConfig != nil {
		if tunnel.GetSaveConfig() {
			directives["SaveConfig"] = "true"
		} else {
			directives["SaveConfig"] = "false"
		}
	}
	setString(directives, "PreUp", tunnel.GetPreUp())
	setString(directives, "PostUp", tunnel.GetPostUp())
	setString(directives, "PreDown", tunnel.GetPreDown())
	setString(directives, "PostDown", tunnel.GetPostDown())

	peers := buildPeers(tunnel.GetPeers())
	return &ast.Interface{
		Name:       tunnel.GetName(),
		Directives: directives,
		Peers:      peers,
	}
}

func buildPeers(peers []*wireguardv1.WireguardPeer) []*ast.Peer {
	var result []*ast.Peer
	for _, peer := range peers {
		if peer == nil {
			continue
		}
		directives := make(map[string]string)
		setString(directives, "AllowedIPs", peer.GetAllowedIps())
		setString(directives, "PublicKey", peer.GetPublicKey())
		if peer.GetPresharedKey() != "" {
			directives["PreSharedKey"] = peer.GetPresharedKey()
		}
		if host := peer.GetEndpointHost(); host != "" {
			endpoint := host
			if port := peer.GetEndpointPort(); port != 0 {
				endpoint = fmt.Sprintf("%s:%d", host, port)
			}
			directives["Endpoint"] = endpoint
		}
		result = append(result, &ast.Peer{
			Name:       peer.GetPublicKey(),
			Directives: directives,
		})
	}
	return result
}

func setString(dest map[string]string, key, value string) {
	if strings.TrimSpace(value) == "" {
		return
	}
	dest[key] = value
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
