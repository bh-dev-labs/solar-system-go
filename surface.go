package main

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"sync"
)

// Procedural equirectangular surface textures, 1024 x 512, generated once
// per style and cached. They are illustrative, not photographic.

const texW, texH = 1024, 512

var (
	texMu    sync.Mutex
	texCache = map[string][]byte{}
)

func surfacePNG(style string) []byte {
	texMu.Lock()
	defer texMu.Unlock()
	if b, ok := texCache[style]; ok {
		return b
	}
	img := image.NewRGBA(image.Rect(0, 0, texW, texH))
	for y := 0; y < texH; y++ {
		lat := (0.5 - float64(y)/texH) * math.Pi
		for x := 0; x < texW; x++ {
			lon := float64(x) / texW * 2 * math.Pi
			// Sample noise on the sphere so the texture wraps seamlessly.
			px := math.Cos(lat) * math.Cos(lon)
			py := math.Cos(lat) * math.Sin(lon)
			pz := math.Sin(lat)
			img.Set(x, y, shade(style, px, py, pz, lat, lon))
		}
	}
	var buf bytes.Buffer
	png.Encode(&buf, img)
	texCache[style] = buf.Bytes()
	return buf.Bytes()
}

func shade(style string, x, y, z, lat, lon float64) color.RGBA {
	switch style {
	case "sun":
		g := fbm(x*18, y*18, z*18, 4)
		v := 0.75 + 0.25*g
		return rgb(255*v, 200*v+20, 90*v)
	case "craters":
		base := 0.45 + 0.25*fbm(x*3, y*3, z*3, 4)
		c := craters(x, y, z, 24, 0.9) * 0.35
		v := clamp(base-c, 0, 1)
		return rgb(190*v+30, 185*v+30, 175*v+30)
	case "venus":
		b := fbm(x*4+lon*0.5, y*4, z*6, 5)
		s := 0.5 + 0.5*math.Sin(lat*9+b*4)
		v := 0.7 + 0.2*b + 0.1*s
		return rgb(235*v, 205*v, 140*v)
	case "earth":
		h := fbm(x*2.2, y*2.2, z*2.2, 6)
		ice := math.Abs(lat) > 1.25+0.08*fbm(x*8, y*8, z*8, 2)
		cloud := fbm(x*5+1, y*5, z*5+3, 4)
		var c color.RGBA
		switch {
		case ice:
			c = rgb(235, 240, 245)
		case h > 0.12:
			g := clamp((h-0.12)*4, 0, 1)
			if math.Abs(lat) < 0.5 && fbm(x*6, y*6, z*6, 3) > 0.1 {
				c = rgb(150+60*g, 130+30*g, 70) // deserts
			} else {
				c = rgb(60+90*g, 120+40*g, 50+30*g)
			}
		case h > 0.06:
			c = rgb(40, 110, 170) // shelf
		default:
			c = rgb(18, 60, 130)
		}
		if cloud > 0.22 {
			w := clamp((cloud-0.22)*5, 0, 0.85)
			c = mix(c, rgb(255, 255, 255), w)
		}
		return c
	case "mars":
		h := fbm(x*3, y*3, z*3, 5)
		v := 0.55 + 0.35*h
		if math.Abs(lat) > 1.3+0.05*fbm(x*9, y*9, z*9, 2) {
			return rgb(240, 235, 230)
		}
		c := craters(x, y, z, 18, 0.7) * 0.25
		v = clamp(v-c, 0.2, 1)
		if h < -0.1 {
			return rgb(120*v+30, 60*v+10, 40*v)
		}
		return rgb(215*v+20, 110*v+15, 60*v)
	case "jupiter":
		band := math.Sin(lat*11 + 0.8*fbm(x*3, y*3, z*3, 3)) // belts and zones
		t := fbm(x*6+lon*0.3, y*6, z*2, 4)
		v := 0.72 + 0.14*band + 0.1*t
		c := rgb(225*v, 190*v, 150*v)
		if band < -0.3 {
			c = rgb(190*v, 130*v, 90*v)
		}
		// Great Red Spot at 22° south.
		d := math.Hypot((lon-4.2)*math.Cos(lat)/0.42, (lat+0.38)/0.2)
		if d < 1 {
			c = mix(rgb(200, 90, 60), c, d*d)
		}
		return c
	case "saturn":
		band := math.Sin(lat*9 + 0.5*fbm(x*2, y*2, z*2, 3))
		v := 0.8 + 0.08*band + 0.05*fbm(x*5, y*5, z*2, 3)
		return rgb(230*v, 205*v, 150*v)
	case "uranus":
		v := 0.9 + 0.04*math.Sin(lat*7) + 0.03*fbm(x*3, y*3, z*3, 3)
		return rgb(160*v, 215*v, 225*v)
	case "neptune":
		band := math.Sin(lat*6 + fbm(x*3, y*3, z*3, 3))
		v := 0.8 + 0.1*band + 0.05*fbm(x*6, y*6, z*2, 3)
		c := rgb(70*v, 110*v, 230*v)
		d := math.Hypot((lon-2.0)*math.Cos(lat)/0.35, (lat+0.35)/0.18)
		if d < 1 {
			c = mix(rgb(25, 40, 120), c, d)
		}
		return c
	case "pluto":
		h := fbm(x*3, y*3, z*3, 5)
		v := 0.6 + 0.3*h
		c := rgb(210*v, 170*v, 130*v)
		// Sputnik Planitia, the bright nitrogen-ice heart, near the equator.
		d := math.Hypot((lon-3.1)*math.Cos(lat)/0.5, (lat-0.2)/0.45)
		if d < 1 {
			c = mix(rgb(240, 235, 225), c, d*d*d)
		}
		if h < -0.2 {
			c = rgb(70*v+20, 40*v+10, 30*v)
		}
		return c
	case "comet":
		base := 0.3 + 0.25*fbm(x*4, y*4, z*4, 5)
		lobes := 0.1 * math.Cos(lon*2)
		v := clamp(base+lobes-craters(x, y, z, 30, 0.8)*0.2, 0.05, 0.8)
		return rgb(90*v+10, 85*v+10, 80*v+10)
	}
	return rgb(128, 128, 128)
}

// hash-based value noise on R^3.
func hash3(ix, iy, iz int) float64 {
	n := uint32(ix)*73856093 ^ uint32(iy)*19349663 ^ uint32(iz)*83492791
	n = (n ^ (n >> 13)) * 1274126177
	n ^= n >> 16
	return float64(n&0xffff)/0xffff*2 - 1
}

func smooth(t float64) float64 { return t * t * (3 - 2*t) }

func noise(x, y, z float64) float64 {
	ix, iy, iz := math.Floor(x), math.Floor(y), math.Floor(z)
	fx, fy, fz := smooth(x-ix), smooth(y-iy), smooth(z-iz)
	i, j, k := int(ix), int(iy), int(iz)
	lerp := func(a, b, t float64) float64 { return a + (b-a)*t }
	return lerp(
		lerp(lerp(hash3(i, j, k), hash3(i+1, j, k), fx), lerp(hash3(i, j+1, k), hash3(i+1, j+1, k), fx), fy),
		lerp(lerp(hash3(i, j, k+1), hash3(i+1, j, k+1), fx), lerp(hash3(i, j+1, k+1), hash3(i+1, j+1, k+1), fx), fy),
		fz)
}

func fbm(x, y, z float64, oct int) float64 {
	sum, amp, norm := 0.0, 0.5, 0.0
	for o := 0; o < oct; o++ {
		sum += amp * noise(x, y, z)
		norm += amp
		x, y, z = x*2.03+7, y*2.01+3, z*1.99+5
		amp *= 0.5
	}
	return sum / norm
}

// craters returns a 0..1 darkening from a scattered field of bowl shapes.
func craters(x, y, z, scale, density float64) float64 {
	px, py, pz := x*scale, y*scale, z*scale
	ix, iy, iz := math.Floor(px), math.Floor(py), math.Floor(pz)
	best := 0.0
	for dx := -1; dx <= 1; dx++ {
		for dy := -1; dy <= 1; dy++ {
			for dz := -1; dz <= 1; dz++ {
				cx, cy, cz := int(ix)+dx, int(iy)+dy, int(iz)+dz
				if (hash3(cx, cy, cz)+1)/2 > density {
					continue
				}
				ox := (hash3(cx+11, cy, cz) + 1) / 2
				oy := (hash3(cx, cy+11, cz) + 1) / 2
				oz := (hash3(cx, cy, cz+11) + 1) / 2
				r := 0.15 + 0.3*(hash3(cx+5, cy+5, cz+5)+1)/2
				d := math.Sqrt(sq(px-(float64(cx)+ox)) + sq(py-(float64(cy)+oy)) + sq(pz-(float64(cz)+oz)))
				if d < r {
					t := d / r
					rim := 1 - math.Abs(t-0.85)*6
					v := (1 - t*t) - clamp(rim, 0, 1)*0.8
					if v > best {
						best = v
					}
				}
			}
		}
	}
	return clamp(best, 0, 1)
}

func sq(v float64) float64 { return v * v }

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func rgb(r, g, b float64) color.RGBA {
	return color.RGBA{uint8(clamp(r, 0, 255)), uint8(clamp(g, 0, 255)), uint8(clamp(b, 0, 255)), 255}
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		uint8(float64(a.R)*(1-t) + float64(b.R)*t),
		uint8(float64(a.G)*(1-t) + float64(b.G)*t),
		uint8(float64(a.B)*(1-t) + float64(b.B)*t), 255}
}
