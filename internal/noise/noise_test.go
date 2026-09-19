package noise

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"
)

func BenchmarkNoise(b *testing.B) {
	var res float64
	for b.Loop() {
		res = PerlinNoise2D(100.0/16, 100.0/16, 0.5, 5)
	}
	if res != -0.1526025750067841 {
		b.Error(fmt.Errorf("Expect: \n%f\ngot:\n%f\n", res, -0.1526025750067841))
	}
}

func BenchmarkNoiseImage(b *testing.B) {
	width, height := 2560, 1440

	for b.Loop() {
		img := image.NewGray(image.Rect(0, 0, width, height))
		for y := range height {
			for x := range width {
				value := PerlinNoise2D(float64(x)/16, float64(y)/16, 0.5, 5)
				gray := uint8(math.Max(0, math.Min(255, (value+1)/2*255)))
				img.SetGray(x, y, color.Gray{Y: gray})
			}
		}
	}
}

func BenchmarkAsyncChunkNoiseImage(b *testing.B) {
	width, height := 2560, 1440

	for b.Loop() {
		res := AsyncChunkPerlinNoise2D(2560, 1440, 0.5, 5)
		img := image.NewGray(image.Rect(0, 0, width, height))
		for y := range height {
			noiseRow := res[y]
			for x := range width {
				value := noiseRow[x]
				gray := uint8(math.Max(0, math.Min(255, (value+1)/2*255)))
				img.SetGray(x, y, color.Gray{Y: gray})
			}
		}

	}
}

func BenchmarkAsyncFlatNoiseImage(b *testing.B) {
	width, height := 2560, 1440

	for b.Loop() {
		res := AsyncFlatPerlinNoise2D(width, height, 0.5, 5)
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
