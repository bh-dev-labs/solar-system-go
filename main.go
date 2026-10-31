// Command solarsystem serves an educational solar-system model.
//
// Go does the physics: it evaluates JPL Keplerian elements for every body at
// the requested Julian date and renders procedural surface textures. The
// browser only draws what it is told.
package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//go:embed static
var staticFS embed.FS

type bodyState struct {
	Name     string  `json:"n"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	Z        float64 `json:"z"`
	R        float64 `json:"r"`  // au from Sun
	V        float64 `json:"v"`  // km/s
	LightMin float64 `json:"lt"` // light minutes from the Sun
	A        float64 `json:"a"`  // semi-major axis, au
}

type state struct {
	JD     float64              `json:"jd"`
	Date   string               `json:"date"`
	Bodies []bodyState          `json:"bodies"`
	Belts  map[string][]float32 `json:"belts"` // flat x,y,z triples
}

func jdToTime(jd float64) time.Time {
	return time.Unix(int64((jd-2440587.5)*86400), 0).UTC()
}

func timeToJD(t time.Time) float64 {
	return float64(t.Unix())/86400 + 2440587.5
}

func computeState(jd float64, wantBelts bool) state {
	s := state{JD: jd, Date: jdToTime(jd).Format("2006-01-02 15:04 UTC")}
	for _, b := range bodies {
		if b.Kind == "star" {
			s.Bodies = append(s.Bodies, bodyState{Name: b.Name})
			continue
		}
		p, a := b.position(jd)
		r := p.Len()
		s.Bodies = append(s.Bodies, bodyState{
			Name: b.Name, X: p.X, Y: p.Y, Z: p.Z, R: r, A: a,
			V: visViva(r, a), LightMin: r * lightSecs / 60,
		})
	}
	if wantBelts {
		s.Belts = map[string][]float32{}
		for _, belt := range belts {
			pts := make([]float32, 0, len(belt.Elems)*3)
			for _, el := range belt.Elems {
				p := el.At(jd)
				pts = append(pts, float32(p.X), float32(p.Y), float32(p.Z))
			}
			s.Belts[belt.Name] = pts
		}
	}
	return s
}

// orbitPath samples one full orbit so the browser can draw the ellipse.
func orbitPath(b *Body, jd float64, n int) [][3]float64 {
	if b.Kind == "star" {
		return nil
	}
	var period float64
	if b.planet != nil {
		_, a := b.planet.At(jd)
		period = 360 / meanMotion(a)
	} else {
		period = 360 / b.small.N
	}
	// Sample uniformly in mean anomaly, which is what time does.
	path := make([][3]float64, 0, n+1)
	for i := 0; i <= n; i++ {
		p, _ := b.position(jd - period/2 + period*float64(i)/float64(n))
		path = append(path, [3]float64{p.X, p.Y, p.Z})
	}
	return path
}

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "listen address")
	flag.Parse()

	mux := http.NewServeMux()
	sub, _ := fs.Sub(staticFS, "static")
	files := http.FileServer(http.FS(sub))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		files.ServeHTTP(w, r)
	})

	mux.HandleFunc("/api/bodies", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, bodies)
	})
	mux.HandleFunc("/api/state", func(w http.ResponseWriter, r *http.Request) {
		jd := timeToJD(time.Now())
		if q := r.URL.Query().Get("jd"); q != "" {
			if v, err := strconv.ParseFloat(q, 64); err == nil {
				jd = v
			}
		}
		writeJSON(w, computeState(jd, r.URL.Query().Get("belts") != "0"))
	})
	mux.HandleFunc("/api/orbits", func(w http.ResponseWriter, r *http.Request) {
		jd := timeToJD(time.Now())
		if q := r.URL.Query().Get("jd"); q != "" {
			if v, err := strconv.ParseFloat(q, 64); err == nil {
				jd = v
			}
		}
		out := map[string][][3]float64{}
		for _, b := range bodies {
			if p := orbitPath(b, jd, 360); p != nil {
				out[b.Name] = p
			}
		}
		writeJSON(w, out)
	})
	mux.HandleFunc("/surface/", func(w http.ResponseWriter, r *http.Request) {
		style := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/surface/"), ".png")
		w.Header().Set("Content-Type", "image/png")
		w.Header().Set("Cache-Control", "public, max-age=86400")
		w.Write(surfacePNG(style))
	})

	log.Printf("solar system at http://%s", *addr)
	log.Fatal(http.ListenAndServe(*addr, mux))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, fmt.Sprintf("encode: %v", err), 500)
	}
}
