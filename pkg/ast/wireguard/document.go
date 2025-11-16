package wireguard

import "io/fs"

// Document 表示一组 WireGuard 配置。
type Document struct {
	Interfaces []*Interface
	Files      []File
}

// File 描述需要写出的附加文件。
type File struct {
	Path     string
	Mode     fs.FileMode
	Contents []byte
}

// Interface 对应 "[Interface]" 块。
type Interface struct {
	Name       string
	Directives map[string]string
	Peers      []*Peer
}

// Peer 对应 "[Peer]" 块。
type Peer struct {
	Name       string
	Directives map[string]string
}
