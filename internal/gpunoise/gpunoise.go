package gpunoise

import (
	"context"
	"fmt"
	"sync"
	"time"
	"unsafe"

	"github.com/Nikolay-Yakunin/noise/static"
	"github.com/gogpu/gputypes"
	"github.com/gogpu/wgpu"
	_ "github.com/gogpu/wgpu/hal/allbackends"
)

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
			WGSL:  static.Shader,
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
