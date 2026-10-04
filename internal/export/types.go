package export

// ABOUTME: Defines export format types and the list of supported export formats.
// ABOUTME: Includes text formats (TXT, GO, JS, PY, RS, SH) and binary formats (PNG).

// ExportFormat represents a supported export format
type ExportFormat struct {
	Name        string // Canonical key (e.g., "TXT", "PNG")
	Extension   string
	Description string
	IsBinary    bool // True for binary formats like PNG that need []byte handling
}

// Available export formats
var SupportedFormats = []ExportFormat{
	{
		Name:        "PNG",
		Extension:   ".png",
		Description: "PNG image (transparent background)",
		IsBinary:    true,
	},
	{
		Name:        "SVG",
		Extension:   ".svg",
		Description: "SVG vector image (transparent background)",
	},
	{
		Name:        "TXT",
		Extension:   ".txt",
		Description: "Plain text file",
	},
	{
		Name:        "GO",
		Extension:   ".go",
		Description: "Go source code",
	},
	{
		Name:        "JS",
		Extension:   ".js",
		Description: "JavaScript source code",
	},
	{
		Name:        "PY",
		Extension:   ".py",
		Description: "Python source code",
	},
	{
		Name:        "RS",
		Extension:   ".rs",
		Description: "Rust source code",
	},
	{
		Name:        "SH",
		Extension:   ".sh",
		Description: "Bash script",
	},
}
