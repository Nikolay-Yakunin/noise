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
			value := noise.PerlinNoise2D(float64(x)/16, float64(y)/16, per, oct)
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
