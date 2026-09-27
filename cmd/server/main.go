package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"log"
	"math"
	"net/http"
	"net/url"
	"strconv"

	"github.com/Nikolay-Yakunin/noise/internal/gpunoise"
	"github.com/Nikolay-Yakunin/noise/internal/noise"
	"github.com/Nikolay-Yakunin/noise/static"
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

	http.HandleFunc("/noiseValue", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		q := r.URL.Query()
		opts := valueOprtionsFromQuery(q)

		values := noise.AsyncFlatValueNoise2D(opts.width, opts.height, opts.scale, opts.per, opts.oct)

		writeGrayPNG(w, opts.width, opts.height, values)
	})

	http.HandleFunc("/noiseValueGPU", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		q := r.URL.Query()
		opts := valueOprtionsFromQuery(q)

		values, err := gpunoise.GPUValueNoise2D(opts.width, opts.height, float64(opts.scale), opts.per, opts.oct)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return

		}
		writeGrayPNG(w, opts.width, opts.height, values)
	})

	http.HandleFunc("/noisePerlin", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		opts, err := perlinOptionsFromQuery(r.URL.Query())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		p := noise.NewPerlin(opts.alpha, opts.beta, opts.octaves, opts.seed)
		values := noise.AsyncFlatPerlin2D(p, opts.width, opts.height, opts.scale)

		writeGrayPNG(w, opts.width, opts.height, values)
	})

	http.HandleFunc("/noisePerlin3d", func(w http.ResponseWriter, r *http.Request) {
		opts, err := perlinOptionsFromQuery(r.URL.Query())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		p := noise.NewPerlin(opts.alpha, opts.beta, opts.octaves, opts.seed)
		values := noise.AsyncFlatPerlin3D(p, opts.width, opts.height, opts.scale, opts.z)

		writeGrayPNG(w, opts.width, opts.height, values)
	})

	http.HandleFunc("/view3d", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, "%s", static.View3D)
	})

	http.HandleFunc("/shader_raymarch.wgsl", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		fmt.Fprintf(w, "%s", static.ShaderRaymarch)
	})

	http.HandleFunc("/perlinTables.bin", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")

		seed := int64(1)
		if n, err := strconv.ParseInt(r.URL.Query().Get("seed"), 10, 64); err == nil {
			seed = n
		}

		p := noise.NewPerlin(2, 2, 1, seed)
		perm, grads := p.Tables3D()

		w.Write(gpunoise.EncodePerlinTables(perm, grads))
	})

	log.Fatal(http.ListenAndServe(":9090", nil))
}

type valueOptions struct {
	width, height, scale, oct int
	per                       float64
}

func valueOprtionsFromQuery(q url.Values) valueOptions {
	opts := valueOptions{
		width:  2560,
		height: 1440,
		scale:  16,
		oct:    5,
		per:    0.5,
	}

	if n, err := strconv.Atoi(q.Get("width")); err == nil && n > 0 {
		opts.width = n
	}
	if n, err := strconv.Atoi(q.Get("height")); err == nil && n > 0 {
		opts.height = n
	}
	if n, err := strconv.Atoi(q.Get("scale")); err == nil && n > 0 {
		opts.scale = n
	}
	if n, err := strconv.ParseFloat(q.Get("per"), 64); err == nil && n > 0 {
		opts.per = n
	}
	if n, err := strconv.Atoi(q.Get("oct")); err == nil && n > 0 {
		opts.oct = n
	}
	return opts
}

type perlinOptions struct {
	width, height, scale int
	alpha, beta          float64
	octaves              int32
	seed                 int64
	z                    float64
}

func perlinOptionsFromQuery(q url.Values) (perlinOptions, error) {
	opts := perlinOptions{
		width:   2560,
		height:  1440,
		scale:   16,
		alpha:   2,
		beta:    2,
		octaves: 5,
		seed:    1,
		z:       0.5,
	}

	if n, err := strconv.Atoi(q.Get("width")); err == nil && n > 0 {
		opts.width = n
	}
	if n, err := strconv.Atoi(q.Get("height")); err == nil && n > 0 {
		opts.height = n
	}
	if n, err := strconv.Atoi(q.Get("oct")); err == nil && n > 0 {
		opts.octaves = int32(n)
	}
	if n, err := strconv.ParseInt(q.Get("seed"), 10, 64); err == nil {
		opts.seed = n
	}
	if f, err := strconv.ParseFloat(q.Get("alpha"), 64); err == nil {
		if f <= 1 {
			return opts, fmt.Errorf("alpha must be > 1, got %v", f)
		}
		opts.alpha = f
	}
	if f, err := strconv.ParseFloat(q.Get("beta"), 64); err == nil && f > 0 {
		opts.beta = f
	}
	if n, err := strconv.Atoi(q.Get("scale")); err == nil {
		if n <= 0 {
			return opts, fmt.Errorf("scale must be > 0, got %d", n)
		}
		opts.scale = n
	}
	if f, err := strconv.ParseFloat(q.Get("z"), 64); err == nil {
		if f < 0 {
			return opts, fmt.Errorf("z must be >= 0, got %v", f)
		}
		opts.z = f
	}

	return opts, nil
}

func writeGrayPNG(w http.ResponseWriter, width, height int, values []float64) {
	w.Header().Set("Content-Type", "image/png")

	img := image.NewGray(image.Rect(0, 0, width, height))
	for y := range height {
		offset := y * width
		noiseRow := values[offset : offset+width]
		for x := range width {
			value := noiseRow[x]
			gray := uint8(math.Max(0, math.Min(255, (value+1)/2*255)))
			img.SetGray(x, y, color.Gray{Y: gray})
		}
	}

	png.Encode(w, img)
}
