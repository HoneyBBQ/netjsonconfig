package netjsonconfig

import (
	"io/fs"
	"time"
)

// Package 表示单个配置包（如 UCI 的 system/network/wireless）。
type Package struct {
	Name    string // 包名（如 "system", "network"）
	Content []byte // 配置内容（不包含 "package" 声明行）
}

// File 表示附加文件（如证书、脚本等）。
type File struct {
	Path    string      // 文件路径
	Content []byte      // 文件内容
	Mode    fs.FileMode // 文件权限
}

// Metadata 存储配置生成的元信息。
type Metadata struct {
	Format    string            // 格式标识（"uci", "openvpn", "wireguard", "vxlan"）
	Backend   string            // 后端名称
	Generated time.Time         // 生成时间
	Version   string            // 版本号
	Custom    map[string]string // 自定义元数据
}

// Bundle 表示完整的配置输出。
type Bundle struct {
	Packages []Package  // 配置包列表
	Files    []File     // 附加文件列表
	Metadata Metadata   // 元信息
}

// NewBundle 创建一个空的 Bundle。
func NewBundle(format, backend string) *Bundle {
	return &Bundle{
		Packages: make([]Package, 0),
		Files:    make([]File, 0),
		Metadata: Metadata{
			Format:    format,
			Backend:   backend,
			Generated: time.Now(),
			Custom:    make(map[string]string),
		},
	}
}

// NativeBundle 已废弃，使用 Bundle 代替。
// 为了向后兼容保留此类型别名。
type NativeBundle = Bundle

// AdditionalFile 已废弃，使用 File 代替。
type AdditionalFile = File
