// Package static is a static files for server
package static

import _ "embed"

//go:embed index.html
var Index string

//go:embed shader.wgsl
var Shader string
