package netjsonconfig

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// Backend 定义双向转换接口，所有协议实现都必须遵循。
type Backend interface {
	// Name 返回 backend 标识，如 "openwrt"、"openvpn"。
	Name() string
	// ToNative 将 NetJSON proto 渲染为目标 DSL。
	ToNative(ctx context.Context, cfg proto.Message, opts RenderOptions) (*Bundle, error)
	// ToNetJSON 将原生 DSL 解析回 NetJSON proto。
	ToNetJSON(ctx context.Context, bundle *Bundle, opts ParseOptions) (proto.Message, error)
}
