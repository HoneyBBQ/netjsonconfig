package vxlan

import wireguardast "github.com/honeybbq/netjsonconfig/pkg/ast/wireguard"

// Document 表示 VXLAN 组合配置，内部复用 WireGuard 文档。
type Document struct {
	Wireguard *wireguardast.Document
	Tunnels   []*Tunnel
}

// Tunnel 描述 VXLAN Tunnels 的元数据。
type Tunnel struct {
	Name    string
	AutoVNI bool
	VNI     uint32
}
