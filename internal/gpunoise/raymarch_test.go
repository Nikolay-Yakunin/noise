package gpunoise

import (
	"math"
	"testing"

	"github.com/Nikolay-Yakunin/noise/static"
	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
)

var raymarchBindGroupLayoutEntries = []gputypes.BindGroupLayoutEntry{
	{Binding: 0, Visibility: wgpu.ShaderStageFragment, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform}},
	{Binding: 1, Visibility: wgpu.ShaderStageFragment, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeReadOnlyStorage}},
}

func TestRaymarchShaderPipeline(t *testing.T) {
	ctx, err := initGPUContext()
	if err != nil {
		t.Skipf("gpu unavailable: %v", err)
	}

	shader, err := ctx.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label: "Raymarch Shader",
		WGSL:  static.ShaderRaymarch,
	})
	if err != nil {
		t.Fatalf("create shader module: %v", err)
	}
	defer shader.Release()

	bgLayout, err := ctx.device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label:   "Raymarch Bind Group Layout",
		Entries: raymarchBindGroupLayoutEntries,
	})
	if err != nil {
		t.Fatalf("create bind group layout: %v", err)
	}
	defer bgLayout.Release()

	layout, err := ctx.device.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{
		Label:            "Raymarch Pipeline Layout",
		BindGroupLayouts: []*wgpu.BindGroupLayout{bgLayout},
	})
	if err != nil {
		t.Fatalf("create pipeline layout: %v", err)
	}
	defer layout.Release()

	pipeline, err := ctx.device.CreateRenderPipeline(&wgpu.RenderPipelineDescriptor{
		Label:  "Raymarch Pipeline",
		Layout: layout,
		Vertex: wgpu.VertexState{Module: shader, EntryPoint: "vsMain"},
		Fragment: &wgpu.FragmentState{
			Module:     shader,
			EntryPoint: "raymarch",
			Targets:    []gputypes.ColorTargetState{{Format: gputypes.TextureFormatBGRA8Unorm}},
		},
		Primitive: gputypes.PrimitiveState{Topology: gputypes.PrimitiveTopologyTriangleList},
	})
	if err != nil {
		t.Fatalf("create render pipeline: %v", err)
	}
	defer pipeline.Release()
}

func TestEncodePerlinTables(t *testing.T) {
	perm := []int32{0, 1, 255, 511}
	grads := [][3]float32{
		{1, 0, 0},
		{0.5, -0.5, 0},
		{0, 0, 1},
		{-1, 0, 0},
	}

	packed := EncodePerlinTables(perm, grads)

	if len(packed) != len(perm)*PerlinTableStride {
		t.Fatalf("packed length = %d, want %d", len(packed), len(perm)*PerlinTableStride)
	}

	for i := range perm {
		entry := readTableEntry(packed, i)
		if got := int32(entry[0]); got != perm[i] {
			t.Errorf("entry %d permutation = %d, want %d", i, got, perm[i])
		}
		for j := range grads[i] {
			if entry[j+1] != grads[i][j] {
				t.Errorf("entry %d gradient[%d] = %v, want %v", i, j, entry[j+1], grads[i][j])
			}
		}
	}
}

func readTableEntry(packed []byte, index int) [4]float32 {
	var entry [4]float32
	offset := index * PerlinTableStride
	for i := range entry {
		entry[i] = math.Float32frombits(leUint32(packed[offset+i*4:]))
	}
	return entry
}

func leUint32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}
