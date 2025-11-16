package netjsonconfig

import (
	"io/fs"
	"time"
)

// Package represents a single configuration package.
// For UCI backends, each package corresponds to a file under /etc/config/
// (e.g., "system", "network", "wireless").
// For other backends, there's typically only one package containing the main config.
type Package struct {
	Name    string // Package name (e.g., "system", "network")
	Content []byte // Configuration content (excluding "package" declaration line for UCI)
}

// File represents an additional file (certificates, scripts, keys, etc.)
// that should be deployed alongside the main configuration.
type File struct {
	Path    string      // Absolute file path where the file should be placed
	Content []byte      // File content (binary-safe)
	Mode    fs.FileMode // Unix file permissions (e.g., 0644, 0600)
}

// Metadata stores information about how and when the configuration was generated.
type Metadata struct {
	Format    string            // Format identifier ("uci", "openvpn", "wireguard", "vxlan")
	Backend   string            // Backend name that generated this bundle
	Generated time.Time         // Timestamp when the bundle was created
	Version   string            // Optional version tag
	Custom    map[string]string // Extensible metadata for backend-specific information
}

// Bundle represents the complete output of a configuration render operation.
// It contains one or more configuration packages, optional additional files,
// and metadata about the generation process.
type Bundle struct {
	Packages []Package // Configuration packages (one or more depending on backend)
	Files    []File    // Additional files to be deployed (certificates, keys, scripts)
	Metadata Metadata  // Generation metadata
}

// NewBundle creates an empty Bundle with initialized metadata.
// The Generated timestamp is set to the current time.
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

// NativeBundle is deprecated. Use Bundle instead.
// Kept for backward compatibility.
type NativeBundle = Bundle

// AdditionalFile is deprecated. Use File instead.
// Kept for backward compatibility.
type AdditionalFile = File
