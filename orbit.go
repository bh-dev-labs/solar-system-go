package main

import "math"

// Orbital mechanics. Everything here works in heliocentric J2000 ecliptic
// coordinates: x toward the vernal equinox, z toward ecliptic north, units au.

const (
	deg2rad   = math.Pi / 180
	j2000     = 2451545.0        // Julian date of J2000.0 epoch
	auKm      = 149597870.7      // km per au (IAU 2012)
	gmSun     = 1.32712440018e11 // km^3/s^2
	lightSecs = 499.004784       // light travel time for 1 au, seconds
)

// Elements is one osculating element set: angles in degrees, a in au,
// M0 is the mean anomaly at Epoch (JD), N the mean motion in deg/day.
type Elements struct {
	A, E, I, Om, W, M0 float64
	Epoch, N           float64
}

// PlanetElements are the JPL "approximate positions" J2000 values and
// per-century rates (Standish table 1, valid 1800-2050 AD).
type PlanetElements struct {
	A, E, I, L, Wbar, Om       float64
	DA, DE, DI, DL, DWbar, DOm float64
}

// Vec3 is a position in au.
type Vec3 struct{ X, Y, Z float64 }

// solveKepler returns the eccentric anomaly E (radians) for mean anomaly M
// (radians) and eccentricity e, using Newton's method as JPL describes it.
func solveKepler(M, e float64) float64 {
	M = math.Mod(M, 2*math.Pi)
	if M < -math.Pi {
		M += 2 * math.Pi
	} else if M > math.Pi {
		M -= 2 * math.Pi
	}
	E := M + e*math.Sin(M)
	if e > 0.8 {
		E = math.Pi
		if M < 0 {
			E = -math.Pi
		}
	}
	for i := 0; i < 60; i++ {
		dM := M - (E - e*math.Sin(E))
		dE := dM / (1 - e*math.Cos(E))
		E += dE
		if math.Abs(dE) < 1e-9 {
			break
		}
	}
	return E
}

// positionFromElements turns a, e, I, w (argument of perihelion), Om and M
// (all angles in degrees) into ecliptic coordinates.
func positionFromElements(a, e, I, w, Om, M float64) Vec3 {
	E := solveKepler(M*deg2rad, e)
	xp := a * (math.Cos(E) - e)
	yp := a * math.Sqrt(1-e*e) * math.Sin(E)
	cw, sw := math.Cos(w*deg2rad), math.Sin(w*deg2rad)
	cO, sO := math.Cos(Om*deg2rad), math.Sin(Om*deg2rad)
	ci, si := math.Cos(I*deg2rad), math.Sin(I*deg2rad)
	return Vec3{
		X: (cw*cO-sw*sO*ci)*xp + (-sw*cO-cw*sO*ci)*yp,
		Y: (cw*sO+sw*cO*ci)*xp + (-sw*sO+cw*cO*ci)*yp,
		Z: (sw*si)*xp + (cw*si)*yp,
	}
}

// At evaluates a fixed element set at Julian date jd.
func (el Elements) At(jd float64) Vec3 {
	M := el.M0 + el.N*(jd-el.Epoch)
	return positionFromElements(el.A, el.E, el.I, el.W, el.Om, M)
}

// At evaluates a planet's secularly varying elements at Julian date jd.
func (p PlanetElements) At(jd float64) (Vec3, float64) {
	T := (jd - j2000) / 36525
	a := p.A + p.DA*T
	e := p.E + p.DE*T
	I := p.I + p.DI*T
	L := p.L + p.DL*T
	wbar := p.Wbar + p.DWbar*T
	Om := p.Om + p.DOm*T
	return positionFromElements(a, e, I, wbar-Om, Om, L-wbar), a
}

// meanMotion is Kepler's third law for a body orbiting the Sun: deg/day.
func meanMotion(a float64) float64 {
	return 360 / (365.256898326 * math.Pow(a, 1.5))
}

// visViva gives orbital speed in km/s at distance r (au) on an orbit with
// semi-major axis a (au).
func visViva(r, a float64) float64 {
	return math.Sqrt(gmSun * (2/(r*auKm) - 1/(a*auKm)))
}

func (v Vec3) Len() float64 { return math.Sqrt(v.X*v.X + v.Y*v.Y + v.Z*v.Z) }
