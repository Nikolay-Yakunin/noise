package gpunoise

import (
	"context"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
	_ "github.com/gogpu/wgpu/hal/allbackends"
)

const wgslShader = `
struct Params {
    width: u32,
    height: u32,
    scale: f32,
    persistence: f32,
    octaves: u32,
}

@group(0) @binding(0) var<uniform> params: Params;
@group(0) @binding(1) var<storage, read_write> output: array<f32>;

fn intNoise(x: i32) -> f32 {
    var n = (x << 13) ^ x;
    let x3 = n * n * n;
    var hash = (x3 * 15731 + 789221) * n + 1376312589;
    hash = hash & 0x7fffffff;
    return 1.0 - f32(hash) / 1073741824.0;
}

fn noise1(x: i32, y: i32) -> f32 {
    return intNoise(x + y * 57);
}

fn smoothNoise(x: i32, y: i32) -> f32 {
    let corners = (
        noise1(x-1, y-1) + noise1(x+1, y-1) + 
        noise1(x-1, y+1) + noise1(x+1, y+1)
    ) / 16.0;
    let sides = (
        noise1(x-1, y) + noise1(x+1, y) + 
        noise1(x, y-1) + noise1(x, y+1)
    ) / 8.0;
    let center = noise1(x, y) / 4.0;
    return corners + sides + center;
}

fn interpolate(a: f32, b: f32, x: f32) -> f32 {
    let ft = x * 3.14159265;
    let f = (1.0 - cos(ft)) * 0.5;
    return a * (1.0 - f) + b * f;
}

fn interpolatedNoise(x: f32, y: f32) -> f32 {
    let integerX = i32(x);
    let fractionalX = x - f32(integerX);
    let integerY = i32(y);
    let fractionalY = y - f32(integerY);
    
    let v1 = smoothNoise(integerX, integerY);
    let v2 = smoothNoise(integerX + 1, integerY);
    let v3 = smoothNoise(integerX, integerY + 1);
    let v4 = smoothNoise(integerX + 1, integerY + 1);
    
    let i1 = interpolate(v1, v2, fractionalX);
    let i2 = interpolate(v3, v4, fractionalX);
    return interpolate(i1, i2, fractionalY);
}

@compute @workgroup_size(16, 16)
fn main(@builtin(global_invocation_id) id: vec3<u32>) {
    let x = id.x;
    let y = id.y;
    
    if (x >= params.width || y >= params.height) {
        return;
    }
    
    let px = f32(x) / params.scale;
    let py = f32(y) / params.scale;
    
    var total: f32 = 0.0;
    var frequency: f32 = 1.0;
    var amplitude: f32 = 1.0;
    
    for (var i: u32 = 0; i < params.octaves; i = i + 1) {
        total = total + interpolatedNoise(px * frequency, py * frequency) * amplitude;
        frequency = frequency * 2.0;
        amplitude = amplitude * params.persistence;
    }
    
    let idx = y * params.width + x;
    output[idx] = total;
}
`

type Params struct {
	Width       uint32
	Height      uint32
	Scale       float32
	Persistence float32
	Octaves     uint32
}

type GPUContext struct {
	instance       *wgpu.Instance
	adapter        *wgpu.Adapter
	device         *wgpu.Device
	shader         *wgpu.ShaderModule
	bgLayout       *wgpu.BindGroupLayout
	pipelineLayout *wgpu.PipelineLayout
	pipeline       *wgpu.ComputePipeline
}

var (
	gpuCtx     *GPUContext
	gpuCtxOnce sync.Once
	gpuCtxErr  error
)

// initGPUContext инициализирует GPU контекст один раз
func initGPUContext() (*GPUContext, error) {
	gpuCtxOnce.Do(func() {
		instance, err := wgpu.CreateInstance(nil)
		if err != nil {
			gpuCtxErr = fmt.Errorf("create instance: %w", err)
			return
		}

		adapter, err := instance.RequestAdapter(&wgpu.RequestAdapterOptions{
			PowerPreference: wgpu.PowerPreferenceHighPerformance,
		})
		if err != nil {
			instance.Release()
			gpuCtxErr = fmt.Errorf("request adapter: %w", err)
			return
		}

		device, err := adapter.RequestDevice(nil)
		if err != nil {
			adapter.Release()
			instance.Release()
			gpuCtxErr = fmt.Errorf("request device: %w", err)
			return
		}

		shader, err := device.CreateShaderModule(&wgpu.ShaderModuleDescriptor{
			Label: "Perlin Noise Shader",
			WGSL:  wgslShader,
		})
		if err != nil {
			device.Release()
			adapter.Release()
			instance.Release()
			gpuCtxErr = fmt.Errorf("create shader: %w", err)
			return
		}

		bgLayout, err := device.CreateBindGroupLayout(&wgpu.BindGroupLayoutDescriptor{
			Label: "Bind Group Layout",
			Entries: []gputypes.BindGroupLayoutEntry{
				{Binding: 0, Visibility: wgpu.ShaderStageCompute, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeUniform}},
				{Binding: 1, Visibility: wgpu.ShaderStageCompute, Buffer: &gputypes.BufferBindingLayout{Type: gputypes.BufferBindingTypeStorage}},
			},
		})
		if err != nil {
			shader.Release()
			device.Release()
			adapter.Release()
			instance.Release()
			gpuCtxErr = fmt.Errorf("create bind group layout: %w", err)
			return
		}

		pipelineLayout, err := device.CreatePipelineLayout(&wgpu.PipelineLayoutDescriptor{
			Label:            "Pipeline Layout",
			BindGroupLayouts: []*wgpu.BindGroupLayout{bgLayout},
		})
		if err != nil {
			bgLayout.Release()
			shader.Release()
			device.Release()
			adapter.Release()
			instance.Release()
			gpuCtxErr = fmt.Errorf("create pipeline layout: %w", err)
			return
		}

		pipeline, err := device.CreateComputePipeline(&wgpu.ComputePipelineDescriptor{
			Label:      "Perlin Noise Pipeline",
			Layout:     pipelineLayout,
			Module:     shader,
			EntryPoint: "main",
		})
		if err != nil {
			pipelineLayout.Release()
			bgLayout.Release()
			shader.Release()
			device.Release()
			adapter.Release()
			instance.Release()
			gpuCtxErr = fmt.Errorf("create pipeline: %w", err)
			return
		}

		gpuCtx = &GPUContext{
			instance:       instance,
			adapter:        adapter,
			device:         device,
			shader:         shader,
			bgLayout:       bgLayout,
			pipelineLayout: pipelineLayout,
			pipeline:       pipeline,
		}
	})

	return gpuCtx, gpuCtxErr
}

// CleanupGPU освобождает GPU ресурсы (вызывать при завершении программы)
func CleanupGPU() {
	if gpuCtx != nil {
		gpuCtx.pipeline.Release()
		gpuCtx.pipelineLayout.Release()
		gpuCtx.bgLayout.Release()
		gpuCtx.shader.Release()
		gpuCtx.device.Release()
		gpuCtx.adapter.Release()
		gpuCtx.instance.Release()
		gpuCtx = nil
	}
}

func GPUPerlinNoise2D(width, height int, scale float64, persistence float64, octaves int) ([]float64, error) {
	ctx, err := initGPUContext()
	if err != nil {
		return nil, err
	}

	params := Params{
		Width:       uint32(width),
		Height:      uint32(height),
		Scale:       float32(scale),
		Persistence: float32(persistence),
		Octaves:     uint32(octaves),
	}
	paramsSize := uint64(unsafe.Sizeof(params))

	// Создаем только буферы для данных (это быстро)
	paramsBuffer, err := ctx.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "Params Buffer",
		Size:  paramsSize,
		Usage: wgpu.BufferUsageUniform | wgpu.BufferUsageCopyDst,
	})
	if err != nil {
		return nil, fmt.Errorf("create params buffer: %w", err)
	}
	defer paramsBuffer.Release()

	if err := ctx.device.Queue().WriteBuffer(paramsBuffer, 0, unsafe.Slice((*byte)(unsafe.Pointer(&params)), paramsSize)); err != nil {
		return nil, fmt.Errorf("write params: %w", err)
	}

	bufferSize := uint64(width * height * 4)
	outputBuffer, err := ctx.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "Output Buffer",
		Size:  bufferSize,
		Usage: wgpu.BufferUsageStorage | wgpu.BufferUsageCopySrc,
	})
	if err != nil {
		return nil, fmt.Errorf("create output buffer: %w", err)
	}
	defer outputBuffer.Release()

	stagingBuffer, err := ctx.device.CreateBuffer(&wgpu.BufferDescriptor{
		Label: "Staging Buffer",
		Size:  bufferSize,
		Usage: wgpu.BufferUsageCopyDst | wgpu.BufferUsageMapRead,
	})
	if err != nil {
		return nil, fmt.Errorf("create staging buffer: %w", err)
	}
	defer stagingBuffer.Release()

	bindGroup, err := ctx.device.CreateBindGroup(&wgpu.BindGroupDescriptor{
		Label:  "Bind Group",
		Layout: ctx.bgLayout,
		Entries: []wgpu.BindGroupEntry{
			{Binding: 0, Buffer: paramsBuffer, Size: paramsSize},
			{Binding: 1, Buffer: outputBuffer, Size: bufferSize},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("create bind group: %w", err)
	}
	defer bindGroup.Release()

	encoder, err := ctx.device.CreateCommandEncoder(nil)
	if err != nil {
		return nil, fmt.Errorf("create command encoder: %w", err)
	}

	pass, err := encoder.BeginComputePass(nil)
	if err != nil {
		return nil, fmt.Errorf("begin compute pass: %w", err)
	}
	pass.SetPipeline(ctx.pipeline)
	pass.SetBindGroup(0, bindGroup, nil)
	pass.Dispatch(uint32((width+15)/16), uint32((height+15)/16), 1)

	if err := pass.End(); err != nil {
		return nil, fmt.Errorf("end compute pass: %w", err)
	}

	encoder.CopyBufferToBuffer(outputBuffer, 0, stagingBuffer, 0, bufferSize)

	cmdBuffer, err := encoder.Finish()
	if err != nil {
		return nil, fmt.Errorf("finish command buffer: %w", err)
	}

	if _, err := ctx.device.Queue().Submit(cmdBuffer); err != nil {
		return nil, fmt.Errorf("submit: %w", err)
	}

	ctxTimeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := stagingBuffer.Map(ctxTimeout, wgpu.MapModeRead, 0, bufferSize); err != nil {
		return nil, fmt.Errorf("map buffer: %w", err)
	}
	defer stagingBuffer.Unmap()

	rng, err := stagingBuffer.MappedRange(0, bufferSize)
	if err != nil {
		return nil, fmt.Errorf("mapped range: %w", err)
	}

	data := rng.Bytes()
	result := make([]float64, width*height)
	for i := range width * height {
		result[i] = float64(*(*float32)(unsafe.Pointer(&data[i*4])))
	}

	return result, nil
}
