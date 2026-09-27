package gpunoise

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"regexp"
	"strconv"
	"testing"

	"github.com/Nikolay-Yakunin/noise/static"
	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
)

var perlinBindGroupLayoutEntries = []gputypes.BindGroupLayoutEntry{
	{Binding: 0, Visibility: wgpu.ShaderStageCompute, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform}},
	{Binding: 1, Visibility: wgpu.ShaderStageCompute, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeStorage}},
	{Binding: 2, Visibility: wgpu.ShaderStageCompute, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeReadOnlyStorage}},
	{Binding: 3, Visibility: wgpu.ShaderStageCompute, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeReadOnlyStorage}},
}

func TestPerlinShaderBindingsMatchLayout(t *testing.T) {
	found := map[uint32]bool{}
	for _, m := range regexp.MustCompile(`@group\(0\)\s*@binding\((\d+)\)`).FindAllStringSubmatch(static.ShaderPerlin, -1) {
		n, err := strconv.ParseUint(m[1], 10, 32)
		if err != nil {
			t.Fatalf("bad binding index %q: %v", m[1], err)
		}
		found[uint32(n)] = true
	}

	if len(found) != len(perlinBindGroupLayoutEntries) {
		t.Fatalf("shader declares %d bindings %v, layout has %d", len(found), found, len(perlinBindGroupLayoutEntries))
	}
	for _, e := range perlinBindGroupLayoutEntries {
		if !found[e.Binding] {
			t.Errorf("layout binding %d (%v) has no matching declaration in the shader", e.Binding, e.Buffer.Type)
		}
	}
}

func TestPerlinShaderPipeline(t *testing.T) {
	ctx, err := initGPUContext()
	if err != nil {
		t.Skipf("gpu unavailable: %v", err)
	}

	shader, err := ctx.device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
		Label: "Perlin Noise Shader",
		WGSL:  static.ShaderPerlin,
	})
	if err != nil {
		t.Fatalf("create shader module: %v", err)
	}
	defer shader.Release()

	bgLayout, err := ctx.device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
		Label:   "Perlin Bind Group Layout",
		Entries: perlinBindGroupLayoutEntries,
	})
	if err != nil {
		t.Fatalf("create bind group layout: %v", err)
	}
	defer bgLayout.Release()

	layout, err := ctx.device.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{
		Label:            "Perlin Pipeline Layout",
		BindGroupLayouts: []*wgpu.BindGroupLayout{bgLayout},
	})
	if err != nil {
		t.Fatalf("create pipeline layout: %v", err)
	}
	defer layout.Release()

	pipeline, err := ctx.device.CreateComputePipeline(&wgpu.ComputePipelineDescriptor{
		Label:      "Perlin Noise Pipeline",
		Layout:     layout,
		Module:     shader,
		EntryPoint: "main",
	})
	if err != nil {
		t.Fatalf("create compute pipeline: %v", err)
	}
	defer pipeline.Release()
}

func BenchmarkGPUNoise(b *testing.B) {
	width, height := 2560, 1440

	for b.Loop() {
		res, err := GPUValueNoise2D(width, height, 16, 0.5, 5)
		if err != nil {
			fmt.Println(fmt.Errorf("GPU render erro: %s", err))
		}
		img := image.NewGray(image.Rect(0, 0, width, height))

		for y := range height {
			offset := y * width
			noiseRow := res[offset : offset+width]
			for x := range width {
				value := noiseRow[x]
				gray := uint8(math.Max(0, math.Min(255, (value+1)/2*255)))
				img.SetGray(x, y, color.Gray{Y: gray})
			}
		}
	}
}
