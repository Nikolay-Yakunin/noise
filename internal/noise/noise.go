// Package noise internal logic
package noise

import (
	"math"
	"runtime"
	"sync"
)

func IntNoise1(x int32) float64 {
	x = (x << 13) ^ x
	return (1.0 - float64((x*(x*x*15731+789221)+1376312589)&0x7fffffff)/1073741824.0)
}

func Noise1(x, y int32) float64 {
	return IntNoise1(x + y*57)
}

func SmoothNoise1(x, y int32) float64 {
	corners := (Noise1(x-1, y-1) + Noise1(x+1, y-1) + Noise1(x-1, y+1) + Noise1(x+1, y+1)) / 16
	sides := (Noise1(x-1, y) + Noise1(x+1, y) + Noise1(x, y-1) + Noise1(x, y+1)) / 8
	center := Noise1(x, y) / 4
	return corners + sides + center
}

func Interpolate(a, b, x float64) float64 {
	ft := x * math.Pi
	f := (1 - math.Cos(ft)) * 0.5
	return a*(1-f) + b*f
}

func InterpolatedNoise1(x, y float64) float64 {
	integerX := int32(x)
	fractionalX := x - float64(integerX)
	integerY := int32(y)
	fractionalY := y - float64(integerY)

	v1 := SmoothNoise1(integerX, integerY)
	v2 := SmoothNoise1(integerX+1, integerY)
	v3 := SmoothNoise1(integerX, integerY+1)
	v4 := SmoothNoise1(integerX+1, integerY+1)

	i1 := Interpolate(v1, v2, fractionalX)
	i2 := Interpolate(v3, v4, fractionalX)
	return Interpolate(i1, i2, fractionalY)
}

func PerlinNoise2D(x, y, persistence float64, octaves int) float64 {
	total := 0.0
	frequency := 1.0
	amplitude := 1.0
	for i := 0; i < octaves; i++ {
		total += InterpolatedNoise1(x*frequency, y*frequency) * amplitude
		frequency *= 2.0
		amplitude *= persistence
	}
	return total
}

func AsyncChunkPerlinNoise2D(width, height int, persistence float64, octaves int) [][]float64 {
	grid := make([][]float64, height)
	var wg sync.WaitGroup

	for i := range height {
		grid[i] = make([]float64, width)
	}

	numWorkers := runtime.NumCPU()
	chunkSize := (height + numWorkers - 1) / numWorkers

	for i := range numWorkers {
		startY := i * chunkSize
		endY := min(startY+chunkSize, height)
		if startY >= height {
			break
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for y := start; y < end; y++ {
				row := grid[y]
				for x := range width {
					row[x] = PerlinNoise2D(float64(x)/16, float64(y)/16, persistence, octaves)
				}
			}
		}(startY, endY)
	}

	wg.Wait()
	return grid
}

func AsyncFlatPerlinNoise2D(width, height int, persistence float64, octaves int) []float64 {
	grid := make([]float64, height*width)
	var wg sync.WaitGroup

	numWorkers := runtime.NumCPU()
	chunkSize := (height + numWorkers - 1) / numWorkers

	for i := range numWorkers {
		startY := i * chunkSize
		endY := min(startY+chunkSize, height)
		if startY >= height {
			break
		}

		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for y := start; y < end; y++ {
				offset := y * width
				row := grid[offset : offset+width]
				for x := range width {
					row[x] = PerlinNoise2D(float64(x)/16, float64(y)/16, persistence, octaves)
				}
			}
		}(startY, endY)
	}

	wg.Wait()
	return grid
}
