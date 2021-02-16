package templates

import (
	"embed"
)

// Templates contains template files.
//go:embed pages layout.html
var Templates embed.FS
