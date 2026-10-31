package main

import (
	"math"
	"testing"
)

func TestEarthAtJ2000(t *testing.T) {
	p, a := planetTable["Earth"].At(j2000)
	r := p.Len()
	if math.Abs(r-0.9833) > 0.003 { // Earth is near perihelion in early January
		t.Fatalf("Earth distance at J2000 = %.4f au, want about 0.983", r)
	}
	if v := visViva(r, a); math.Abs(v-30.3) > 0.3 {
		t.Fatalf("Earth speed = %.2f km/s, want about 30.3 near perihelion", v)
	}
	// On 1 Jan the Sun is near ecliptic longitude 280°, so Earth is near 100°.
	lon := math.Atan2(p.Y, p.X) / deg2rad
	if math.Abs(lon-100) > 1.5 {
		t.Fatalf("Earth heliocentric longitude = %.2f°, want about 100°", lon)
	}
}

func TestKeplerThirdLaw(t *testing.T) {
	want := map[string]float64{"Mercury": 88.0, "Venus": 224.7, "Earth": 365.2, "Mars": 687.0,
		"Jupiter": 4331, "Saturn": 10747, "Uranus": 30589, "Neptune": 59800}
	for name, days := range want {
		_, a := planetTable[name].At(j2000)
		got := 360 / meanMotion(a)
		if math.Abs(got-days)/days > 0.01 {
			t.Errorf("%s: period from a=%.3f au is %.0f days, fact sheet says %.0f", name, a, got, days)
		}
	}
}

func TestPerihelionAphelion(t *testing.T) {
	for _, b := range bodies {
		if b.small == nil {
			continue
		}
		min, max := math.Inf(1), 0.0
		period := 360 / b.small.N
		// Sample uniformly in time, plus the exact perihelion instant: a
		// comet with e near 1 spends only hours near q.
		tp := b.small.Epoch - b.small.M0/b.small.N
		for i := 0; i <= 2000; i++ {
			r := b.small.At(b.small.Epoch + period*float64(i)/2000).Len()
			min, max = math.Min(min, r), math.Max(max, r)
		}
		min = math.Min(min, b.small.At(tp).Len())
		q, ad := b.small.A*(1-b.small.E), b.small.A*(1+b.small.E)
		if math.Abs(min-q)/q > 0.01 || math.Abs(max-ad)/ad > 0.01 {
			t.Errorf("%s: sampled %.3f..%.3f au, elements say %.3f..%.3f", b.Name, min, max, q, ad)
		}
	}
}

func TestHalleyPerihelion1986(t *testing.T) {
	// 1P/Halley perihelion was 1986-02-09, JD 2446470.
	r := bodies[len(bodies)-4].small.At(2446469.97).Len()
	if math.Abs(r-0.5749) > 0.002 {
		t.Fatalf("Halley on 1986-02-09 at %.4f au, want 0.5749", r)
	}
}

func TestKirkwoodGapsEmpty(t *testing.T) {
	for _, el := range belts[0].Elems {
		for _, k := range kirkwood {
			if math.Abs(el.A-k.a) < k.half {
				t.Fatalf("asteroid at a=%.3f sits inside the %.3f au gap", el.A, k.a)
			}
		}
	}
}
