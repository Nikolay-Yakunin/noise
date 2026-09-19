package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"net/http"
	"strconv"

	static "github.com/Nikolay-Yakunin/noise/html"
	"github.com/Nikolay-Yakunin/noise/internal/noise"
)

func main() {
	fmt.Println("Hello, Server!")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "%s", static.Index)
	})

	http.HandleFunc("/ping", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "pong")
	})

	http.HandleFunc("/noise", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		per := 0.5
		oct := 5
		width, height := 2560, 1440

		q := r.URL.Query()

		// TODO: Move this bloc to package. used in 2 place, here and in cli
		if n, err := strconv.Atoi(q.Get("width")); err == nil && n > 0 {
			width = n
		}
		if n, err := strconv.Atoi(q.Get("height")); err == nil && n > 0 {
			height = n
		}
		if n, err := strconv.ParseFloat(q.Get("per"), 64); err == nil && n > 0 {
			per = n
		}
		if n, err := strconv.Atoi(q.Get("oct")); err == nil && n > 0 {
			oct = n
		}

		img := image.NewGray(image.Rect(0, 0, width, height))
		for y := range height {
			for x := range width {
				value := noise.PerlinNoise2D(float64(x)/16, float64(y)/16, per, oct)
				gray := uint8(math.Max(0, math.Min(255, (value+1)/2*255)))
				img.SetGray(x, y, color.Gray{Y: gray})
			}
		}
		png.Encode(w, img)
	})

	log.Fatal(http.ListenAndServe(":9090", nil))
}
