package sync

import (
	"time"

	devicev1 "github.com/honeybbq/netjson/gen/go/netjson/device/v1"
)

// VersionedConfig 记录配置版本元数据。
type VersionedConfig struct {
	VersionID string
	Checksum  string
	Timestamp time.Time
	Config    *devicev1.DeviceConfig
}

// ChangeSet 描述一次差异。
type ChangeSet struct {
	Base   *VersionedConfig
	Target *VersionedConfig
	Diff   *DiffResult
}

// DiffResult 暂时只记录原始 JSON，后续可拓展。
type DiffResult struct {
	Added   map[string]any
	Removed map[string]any
	Changed map[string][2]any
}
