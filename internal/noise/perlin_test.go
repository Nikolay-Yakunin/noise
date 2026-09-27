package noise

import (
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"
)

func testPerlin() *Perlin {
	return NewPerlin(2, 2, 5, 1)
}

func TestNewPerlinTables(t *testing.T) {
	p := testPerlin()

	seen := make(map[int32]bool, B)
	for i := range B {
		if p.p[i] < 0 || p.p[i] >= B {
			t.Fatalf("p.p[%d] = %d, out of [0, %d)", i, p.p[i], B)
		}
		if seen[p.p[i]] {
			t.Fatalf("p.p[%d] = %d is not a permutation, value repeats", i, p.p[i])
		}
		seen[p.p[i]] = true
	}

	for i := range B + 2 {
		if p.p[B+i] != p.p[i] {
			t.Errorf("p.p[%d] = %d, want %d", B+i, p.p[B+i], p.p[i])
		}
		if p.g1[B+i] != p.g1[i] {
			t.Errorf("p.g1[%d] = %v, want %v", B+i, p.g1[B+i], p.g1[i])
		}
		if p.g2[B+i] != p.g2[i] {
			t.Errorf("p.g2[%d] = %v, want %v", B+i, p.g2[B+i], p.g2[i])
		}
		if p.g3[B+i] != p.g3[i] {
			t.Errorf("p.g3[%d] = %v, want %v", B+i, p.g3[B+i], p.g3[i])
		}
	}
}

func TestPerlinGradientsNormalized(t *testing.T) {
	p := testPerlin()

	for i := range B {
		if mag := math.Hypot(p.g2[i][0], p.g2[i][1]); math.Abs(mag-1) > 1e-12 {
			t.Errorf("|g2[%d]| = %v, want 1", i, mag)
		}
		if mag := math.Hypot(math.Hypot(p.g3[i][0], p.g3[i][1]), p.g3[i][2]); math.Abs(mag-1) > 1e-12 {
			t.Errorf("|g3[%d]| = %v, want 1", i, mag)
		}
	}
}

func TestPerlinNoiseStats(t *testing.T) {
	for seed := int64(1); seed <= 8; seed++ {
		p := NewPerlin(2, 2, 5, seed)

		cases := []struct {
			name   string
			calc   func(x, y, z float64) float64
			sample func(yield func(x, y, z float64))
		}{
			{
				name: "1D",
				calc: func(x, y, z float64) float64 { return p.Noise1D(x) },
				sample: func(yield func(x, y, z float64)) {
					for i := range 2048 {
						yield(float64(i)/16, 0, 0)
					}
				},
			},
			{
				name: "2D",
				calc: func(x, y, z float64) float64 { return p.Noise2D(x, y) },
				sample: func(yield func(x, y, z float64)) {
					for y := range 512 {
						for x := range 512 {
							yield(float64(x)/16, float64(y)/16, 0)
						}
					}
				},
			},
			{
				name: "3D",
				calc: func(x, y, z float64) float64 { return p.Noise3D(x, y, z) },
				sample: func(yield func(x, y, z float64)) {
					for _, z := range []float64{0.3, 1.7, 3.1} {
						for y := range 128 {
							for x := range 128 {
								yield(float64(x)/16, float64(y)/16, z)
							}
						}
					}
				},
			},
		}

		for _, tc := range cases {
			t.Run(fmt.Sprintf("%s/seed=%d", tc.name, seed), func(t *testing.T) {
				var sum, min, max, count float64
				min, max = math.Inf(1), math.Inf(-1)

				tc.sample(func(x, y, z float64) {
					v := tc.calc(x, y, z)

					if math.IsNaN(v) || math.IsInf(v, 0) {
						t.Fatalf("noise(%v, %v, %v) = %v, want a finite value", x, y, z, v)
					}
					if math.Abs(v) > 2 {
						t.Fatalf("noise(%v, %v, %v) = %v, want |v| < 2", x, y, z, v)
					}

					sum += v
					min = math.Min(min, v)
					max = math.Max(max, v)
					count++
				})

				if max-min < 0.2 {
					t.Errorf("value range = %v, want a non-constant field", max-min)
				}
				if mean := sum / count; math.Abs(mean) > 0.05 {
					t.Errorf("mean = %v, want it close to 0", mean)
				}
				if a, b := tc.calc(0.3, 0.7, 1.1), tc.calc(0.3, 0.7, 1.1); a != b {
					t.Errorf("noise is not deterministic: %v != %v", a, b)
				}
			})
		}
	}
}

func TestPerlinNoiseContinuous(t *testing.T) {
	p := testPerlin()

	for _, tc := range []struct {
		name string
		calc func(x float64) float64
	}{
		{"1D", func(x float64) float64 { return p.Noise1D(x) }},
		{"2D", func(x float64) float64 { return p.Noise2D(x, 0.3) }},
		{"3D", func(x float64) float64 { return p.Noise3D(x, 0.3, 0.7) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			prev := tc.calc(0)
			var maxStep float64

			for i := 1; i <= 4096; i++ {
				cur := tc.calc(float64(i) / 2048)
				maxStep = math.Max(maxStep, math.Abs(cur-prev))
				prev = cur
			}

			if maxStep > 0.02 {
				t.Errorf("max step between neighbours = %v, want a continuous field", maxStep)
			}
		})
	}
}

func TestPerlinSeedChangesNoise(t *testing.T) {
	a := NewPerlin(2, 2, 5, 1)
	b := NewPerlin(2, 2, 5, 2)

	differs := 0
	for y := range 32 {
		for x := range 32 {
			if a.Noise2D(float64(x)/16, float64(y)/16) != b.Noise2D(float64(x)/16, float64(y)/16) {
				differs++
			}
		}
	}

	if differs < 32*32/2 {
		t.Errorf("only %d/%d samples differ between seeds, want a different field", differs, 32*32)
	}
}

func TestPerlinZeroValue(t *testing.T) {
	var p Perlin

	if v := p.Noise1D(1); v != 0 {
		t.Errorf("Noise1D = %v, want 0", v)
	}
	if v := p.Noise2D(1, 2); v != 0 {
		t.Errorf("Noise2D = %v, want 0", v)
	}
	if v := p.Noise3D(1, 2, 3); v != 0 {
		t.Errorf("Noise3D = %v, want 0", v)
	}
}

func TestPerlinNoise3DFallsBackTo2D(t *testing.T) {
	p := testPerlin()

	if got, want := p.Noise3D(1.5, 2.5, -1), p.Noise2D(1.5, 2.5); got != want {
		t.Errorf("Noise3D with z < 0 = %v, want Noise2D = %v", got, want)
	}
}

func TestAsyncFlatPerlin2D(t *testing.T) {
	p := testPerlin()
	const width, height, scale = 61, 37, 16

	grid := AsyncFlatPerlin2D(p, width, height, scale)

	if len(grid) != width*height {
		t.Fatalf("len(grid) = %d, want %d", len(grid), width*height)
	}

	for y := range height {
		for x := range width {
			want := p.Noise2D(float64(x)/scale, float64(y)/scale)
			if got := grid[y*width+x]; got != want {
				t.Fatalf("grid[%d][%d] = %v, want %v", y, x, got, want)
			}
		}
	}
}

func TestAsyncFlatPerlin2DDegenerateSize(t *testing.T) {
	p := testPerlin()

	if got := AsyncFlatPerlin2D(p, 0, 0, 16); len(got) != 0 {
		t.Errorf("len(grid) = %d, want 0", len(got))
	}
	if got := AsyncFlatPerlin2D(p, 8, 0, 16); len(got) != 0 {
		t.Errorf("len(grid) = %d, want 0", len(got))
	}
}

func TestAsyncFlatPerlin3D(t *testing.T) {
	p := testPerlin()
	const width, height, scale = 40, 24, 8
	const z = 12

	grid := AsyncFlatPerlin3D(p, width, height, scale, z)

	if len(grid) != width*height {
		t.Fatalf("len(grid) = %d, want %d", len(grid), width*height)
	}

	var diff float64
	for y := range height {
		for x := range width {
			v := grid[y*width+x]
			if math.IsNaN(v) || math.IsInf(v, 0) {
				t.Fatalf("grid[%d][%d] = %v, want a finite value", y, x, v)
			}
			if want := FlatPerlin3D(p, float64(x), float64(y), z, scale); v != want {
				t.Fatalf("grid[%d][%d] = %v, want %v", y, x, v, want)
			}
			diff += math.Abs(v - p.Noise2D(float64(x)/scale, float64(y)/scale))
		}
	}

	if mean := diff / (width * height); mean < 0.1 {
		t.Errorf("mean |3D - 2D| = %v, want the z slice to actually depend on z", mean)
	}
}

func BenchmarkPerlinNoise1D(b *testing.B) {
	p := testPerlin()

	for b.Loop() {
		p.Noise1D(100.0 / 16)
	}
}

func BenchmarkPerlinNoise2D(b *testing.B) {
	p := testPerlin()

	for b.Loop() {
		p.Noise2D(100.0/16, 100.0/16)
	}
}

func BenchmarkPerlinNoise3D(b *testing.B) {
	p := testPerlin()

	for b.Loop() {
		p.Noise3D(100.0/16, 100.0/16, 100.0/16)
	}
}

func BenchmarkNewPerlin(b *testing.B) {
	for b.Loop() {
		NewPerlin(2, 2, 5, 1)
	}
}

func BenchmarkPerlinNoiseImage(b *testing.B) {
	p := testPerlin()
	width, height := 2560, 1440

	for b.Loop() {
		img := image.NewGray(image.Rect(0, 0, width, height))
		for y := range height {
			for x := range width {
				value := p.Noise2D(float64(x)/16, float64(y)/16)
				gray := uint8(math.Max(0, math.Min(255, (value+1)/2*255)))
				img.SetGray(x, y, color.Gray{Y: gray})
			}
		}
	}
}

func BenchmarkAsyncFlatPerlin2DImage(b *testing.B) {
	p := testPerlin()
	width, height := 2560, 1440

	for b.Loop() {
		res := AsyncFlatPerlin2D(p, width, height, 16)
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
