package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strconv"
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
	for i := range octaves {
		frequency := math.Pow(2, float64(i))
		amplitude := math.Pow(persistence, float64(i))
		total += InterpolatedNoise1(x*frequency, y*frequency) * amplitude
	}
	return total
}

func main() {
	per := 0.5
	oct := 5
	width, height := 2560, 1440
	if len(os.Args) > 1 {
		if n, err := strconv.Atoi(os.Args[1]); err == nil && n > 0 {
			width = n
		}
	}
	if len(os.Args) > 2 {
		if n, err := strconv.Atoi(os.Args[2]); err == nil && n > 0 {
			height = n
		}
	}
	if len(os.Args) > 3 {
		if n, err := strconv.ParseFloat(os.Args[3], 64); err == nil && n > 0 {
			per = n
		}
	}
	if len(os.Args) > 4 {
		if n, err := strconv.Atoi(os.Args[4]); err == nil && n > 0 {
			oct = n
		}
	}

	img := image.NewGray(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			value := PerlinNoise2D(float64(x)/16, float64(y)/16, per, oct)
			gray := uint8(math.Max(0, math.Min(255, (value+1)/2*255)))
			img.SetGray(x, y, color.Gray{Y: gray})
		}
	}

	file, err := os.Create("noise.png")
	if err != nil {
		fmt.Println(err)
		return
	}
	defer file.Close()

	if err := png.Encode(file, img); err != nil {
		fmt.Println(err)
	}
}
