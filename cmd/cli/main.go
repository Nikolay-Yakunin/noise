package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strconv"

	"github.com/Nikolay-Yakunin/noise/internal/noise"
)

func main() {
	per := 0.5
	oct := 5
	width, height, scale := 2560, 1440, 16
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
		if n, err := strconv.Atoi(os.Args[3]); err == nil && n > 0 {
			scale = n
		}
	}
	if len(os.Args) > 4 {
		if n, err := strconv.ParseFloat(os.Args[4], 64); err == nil && n > 0 {
			per = n
		}
	}
	if len(os.Args) > 5 {
		if n, err := strconv.Atoi(os.Args[5]); err == nil && n > 0 {
			oct = n
		}
	}
	res := noise.AsyncFlatPerlinNoise2D(width, height, scale, per, oct)
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
