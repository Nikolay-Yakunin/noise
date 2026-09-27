// Package static is a static files for server
package static

import _ "embed"

//go:embed index.html
var Index string

//go:embed view3d.html
var View3D string

//go:embed shader_value.wgsl
var ShaderValue string

//go:embed shader_perlin.wgsl
var ShaderPerlin string

//go:embed shader_raymarch.wgsl
var ShaderRaymarch string
