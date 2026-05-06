// Package frontend handles the application's user-facing elements, such as
// serving static files and rendering HTML templates
package frontend

import (
	"embed"
)

// AssetsFs holds the content of the embedded FS for application assets
//
//go:embed assets
var AssetsFs embed.FS
