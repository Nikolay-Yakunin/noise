package gpunoise

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"
)

func BenchmarkGPUNoise(b *testing.B) {
	width, height := 2560, 1440

	for b.Loop() {
		res, err := GPUPerlinNoise2D(width, height, 16, 0.5, 5)
		if err != nil {
			fmt.Errorf("GPU render erro: %s", err)
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
