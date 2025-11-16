package openvpn

import "io/fs"

// Document 表示一个 OpenVPN 配置集合，可包含多个实例与附加文件。
type Document struct {
	Instances []*Instance
	Files     []File
}

// File 描述额外需要输出的文件。
type File struct {
	Path     string
	Mode     fs.FileMode
	Contents []byte
}

// Instance 描述单个 OpenVPN 配置。
type Instance struct {
	Name       string
	Directives []Directive
}

// Directive 表示 "key value" 形式的行。
type Directive struct {
	Key      string
	Value    string
	HasValue bool
}
