package netjsonconfig

import "time"

// RenderMode controls the DSL rendering strategy.
// Some backends (like OpenWrt) support multiple syntax versions.
type RenderMode int

const (
	// RenderModeAuto automatically detects the appropriate syntax from input.
	RenderModeAuto RenderMode = iota
	
	// RenderModeDSA forces DSA (Distributed Switch Architecture) syntax for OpenWrt >= 21.
	RenderModeDSA
	
	// RenderModeLegacy forces legacy syntax for OpenWrt <= 19.
	RenderModeLegacy
)

// RenderOptions controls the forward rendering process (NetJSON → DSL).
type RenderOptions struct {
	Mode             RenderMode     // Syntax mode selection
	TemplateContext  map[string]any // Variables for template evaluation
	Strict           bool           // Fail on any warnings if true
	SkipValidation   bool           // Skip schema validation if true
	GenerationTag    string         // Optional tag to include in generated files
	Timeout          time.Duration  // Maximum time allowed for rendering
	IncludeAuxiliary bool           // Whether to include auxiliary files in output
}

// ParseOptions controls the reverse parsing process (DSL → NetJSON).
type ParseOptions struct {
	Mode           RenderMode            // Expected syntax mode
	AllowUnknown   bool                  // Allow unknown fields in input
	AssumeTemplate bool                  // Treat input as template (preserve variables)
	SkipValidation bool                  // Skip schema validation if true
	Timeout        time.Duration         // Maximum time allowed for parsing
	SourceMetadata map[string]string     // Metadata about the source (version, origin, etc.)
	BestEffort     bool                  // Continue parsing on non-fatal errors
}
