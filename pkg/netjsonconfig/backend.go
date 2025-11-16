package netjsonconfig

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// Backend defines the bidirectional conversion interface that all protocol implementations must follow.
// Each backend is responsible for converting between NetJSON (proto.Message) and its native DSL format.
type Backend interface {
	// Name returns the backend identifier (e.g., "openwrt", "openvpn", "wireguard").
	Name() string
	
	// ToNative renders a NetJSON proto message to native DSL format.
	// This is the forward conversion: NetJSON → UCI/OpenVPN/WireGuard config files.
	ToNative(ctx context.Context, cfg proto.Message, opts RenderOptions) (*Bundle, error)
	
	// ToNetJSON parses native DSL back to NetJSON proto message.
	// This is the reverse conversion: UCI/OpenVPN/WireGuard → NetJSON.
	ToNetJSON(ctx context.Context, bundle *Bundle, opts ParseOptions) (proto.Message, error)
}
