package main

import (
	"math"
	"math/rand"
)

// Body is anything we draw. Numbers come from the NASA NSSDC planetary fact
// sheets (updated 18 March 2025) and the JPL Small-Body Database; the
// Source field on each body says which.
type Body struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"` // star, planet, dwarf, asteroid, comet
	Color     string   `json:"color"`
	RadiusKm  float64  `json:"radiusKm"`
	Rings     bool     `json:"rings"`
	Retro     bool     `json:"retrograde"`
	Tilt      float64  `json:"tiltDeg"`
	DayHours  float64  `json:"dayHours"`
	PeriodD   float64  `json:"periodDays"`
	Moons     int      `json:"moons"`
	Surface   string   `json:"surface"`
	Headline  string   `json:"headline"`
	Facts     []string `json:"facts"`
	Source    string   `json:"source"`
	planet    *PlanetElements
	small     *Elements
	semiMajor float64
}

func (b *Body) position(jd float64) (Vec3, float64) {
	switch {
	case b.planet != nil:
		return b.planet.At(jd)
	case b.small != nil:
		return b.small.At(jd), b.small.A
	}
	return Vec3{}, 0
}

// JPL approx_pos.html table 1.
var planetTable = map[string]PlanetElements{
	"Mercury": {0.38709927, 0.20563593, 7.00497902, 252.25032350, 77.45779628, 48.33076593,
		0.00000037, 0.00001906, -0.00594749, 149472.67411175, 0.16047689, -0.12534081},
	"Venus": {0.72333566, 0.00677672, 3.39467605, 181.97909950, 131.60246718, 76.67984255,
		0.00000390, -0.00004107, -0.00078890, 58517.81538729, 0.00268329, -0.27769418},
	"Earth": {1.00000261, 0.01671123, -0.00001531, 100.46457166, 102.93768193, 0.0,
		0.00000562, -0.00004392, -0.01294668, 35999.37244981, 0.32327364, 0.0},
	"Mars": {1.52371034, 0.09339410, 1.84969142, -4.55343205, -23.94362959, 49.55953891,
		0.00001847, 0.00007882, -0.00813131, 19140.30268499, 0.44441088, -0.29257343},
	"Jupiter": {5.20288700, 0.04838624, 1.30439695, 34.39644051, 14.72847983, 100.47390909,
		-0.00011607, -0.00013253, -0.00183714, 3034.74612775, 0.21252668, 0.20469106},
	"Saturn": {9.53667594, 0.05386179, 2.48599187, 49.95424423, 92.59887831, 113.66242448,
		-0.00125060, -0.00050991, 0.00193609, 1222.49362201, -0.41897216, -0.28867794},
	"Uranus": {19.18916464, 0.04725744, 0.77263783, 313.23810451, 170.95427630, 74.01692503,
		-0.00196176, -0.00004397, -0.00242939, 428.48202785, 0.40805281, 0.04240589},
	"Neptune": {30.06992276, 0.00859048, 1.77004347, -55.12002969, 44.96476227, 131.78422574,
		0.00026291, 0.00005105, 0.00035372, 218.45945325, -0.32241464, -0.00508664},
}

// JPL SBDB osculating elements, fetched 2026-10-19 (full precision, trimmed).
func sb(a, e, i, om, w, ma, epoch, per float64) *Elements {
	return &Elements{A: a, E: e, I: i, Om: om, W: w, M0: ma, Epoch: epoch, N: 360 / per}
}

const nssdc = "NASA NSSDC planetary fact sheet, 2025-03-18"
const sbdb = "JPL Small-Body Database, elements as of 2026-10-19"

var bodies = []*Body{
	{Name: "Sun", Kind: "star", Color: "#ffd27a", RadiusKm: 695700, DayHours: 609.12, Surface: "sun",
		Headline: "A G-type main-sequence star holding 99.8% of the solar system's mass.",
		Facts: []string{
			"Mass 1.9884 × 10^30 kg, 333,000 Earths.",
			"Radius 695,700 km, 109 Earth radii.",
			"Photosphere 5,772 K; core 15.7 million K at 162,000 kg/m³.",
			"Luminosity 3.828 × 10^26 W; sunlight takes 8 min 19 s to reach Earth.",
			"Rotates in 25.4 days at 16° latitude; slower toward the poles.",
			"Photosphere is 91% hydrogen and 8.9% helium by atom count; sunspot cycle 11.4 years.",
		}, Source: nssdc},
	{Name: "Mercury", Kind: "planet", Color: "#b5b1a8", RadiusKm: 2439.5, Tilt: 0.034, DayHours: 4222.6, PeriodD: 88.0, Surface: "craters",
		Headline: "Smallest planet, heavily cratered, with a huge iron core.",
		Facts: []string{
			"Diameter 4,879 km; gravity 3.7 m/s²; escape velocity 4.3 km/s.",
			"Mean temperature 167 °C; sunward side reaches 590 to 725 K.",
			"No real atmosphere: a surface-bounded exosphere below 5 × 10^-15 bar, mostly sodium, magnesium, oxygen.",
			"Most eccentric planetary orbit (e = 0.206): 46.0 to 69.8 million km from the Sun.",
			"Spins 3 times for every 2 orbits, so one solar day lasts 176 Earth days.",
			"Weak but real magnetic field, about 1% of Earth's at the surface, offset 0.17 radii north.",
		}, Source: nssdc},
	{Name: "Venus", Kind: "planet", Color: "#e8cf9a", RadiusKm: 6052, Tilt: 177.4, Retro: true, DayHours: 2802.0, PeriodD: 224.7, Surface: "venus",
		Headline: "Earth's twin in size, hidden under a crushing carbon-dioxide greenhouse.",
		Facts: []string{
			"Diameter 12,104 km; gravity 8.9 m/s²; density 5,243 kg/m³.",
			"Surface 464 °C (737 K) day and night, hotter than Mercury; surface pressure 92 bar.",
			"Atmosphere 96.5% CO₂, 3.5% N₂, with 150 ppm sulfur dioxide; surface air density 65 kg/m³.",
			"Rotates backwards once every 243 days, so its day (117 Earth days) is longer than its 225-day year.",
			"Nearly circular orbit, e = 0.007, the roundest of any planet.",
			"Surface winds only 0.3 to 1 m/s; no detected internal magnetic field.",
		}, Source: nssdc},
	{Name: "Earth", Kind: "planet", Color: "#4f8fdc", RadiusKm: 6378, Tilt: 23.4, DayHours: 24.0, PeriodD: 365.2, Moons: 1, Surface: "earth",
		Headline: "The only known world with liquid surface oceans and life.",
		Facts: []string{
			"Diameter 12,756 km; mass 5.97 × 10^24 kg; density 5,514 kg/m³, the densest planet.",
			"Mean temperature 15 °C; surface pressure 1,014 mb.",
			"Dry air is 78.08% N₂, 20.95% O₂, 0.93% Ar and 420 ppm CO₂.",
			"Axial tilt 23.4° drives the seasons; orbit e = 0.017, 147.1 to 152.1 million km.",
			"Escape velocity 11.19 km/s; dipole field 0.306 gauss at the equator shields the surface.",
			"One large moon, 1,737 km in radius, about 384,400 km away.",
		}, Source: nssdc},
	{Name: "Mars", Kind: "planet", Color: "#d0603a", RadiusKm: 3396, Tilt: 25.2, DayHours: 24.7, PeriodD: 687.0, Moons: 2, Surface: "mars",
		Headline: "A cold desert with the solar system's tallest volcano and deepest canyon.",
		Facts: []string{
			"Diameter 6,792 km; gravity 3.7 m/s²; escape velocity 5.03 km/s.",
			"Mean temperature -65 °C; Viking 1 site swung from -89 to -31 °C each day.",
			"Thin air, 6.36 mb mean pressure: 95.1% CO₂, 2.59% N₂, 1.94% Ar, 0.16% O₂.",
			"Day 24.7 h and tilt 25.2°, both close to Earth's, so Mars has seasons.",
			"Dust-storm winds 17 to 30 m/s; iron oxide dust gives the red colour.",
			"Two tiny moons: Phobos (13 km) at 9,378 km and Deimos (7.8 km) at 23,459 km.",
		}, Source: nssdc},
	{Name: "Jupiter", Kind: "planet", Color: "#d9b27c", RadiusKm: 71492, Rings: true, Tilt: 3.1, DayHours: 9.9, PeriodD: 4331, Moons: 95, Surface: "jupiter",
		Headline: "The giant: 318 Earth masses, a 9.9-hour day and a storm older than telescopes.",
		Facts: []string{
			"Diameter 142,984 km; mass 1,898 × 10^24 kg, 2.5 times all the other planets combined.",
			"Atmosphere 89.8% H₂ and 10.2% He; -108 °C at the 1-bar level.",
			"Fastest rotation of any planet, 9.9 h, which flattens it visibly.",
			"Winds up to 150 m/s within 30° of the equator.",
			"Dipole field 4.3 gauss-R³, tilted 9.4°, the strongest planetary magnetic field.",
			"95 known moons including Io, Europa, Ganymede and Callisto; a faint dusty ring.",
		}, Source: nssdc},
	{Name: "Saturn", Kind: "planet", Color: "#e3cf9c", RadiusKm: 60268, Rings: true, Tilt: 26.7, DayHours: 10.7, PeriodD: 10747, Moons: 274, Surface: "saturn",
		Headline: "Less dense than water, wrapped in the brightest ring system.",
		Facts: []string{
			"Diameter 120,536 km; density 687 kg/m³, the only planet lighter than water.",
			"Atmosphere 96.3% H₂ and 3.25% He; -139 °C at 1 bar.",
			"Winds reach 400 m/s near the equator, among the fastest in the solar system.",
			"274 known moons, the most of any planet; Titan is the only moon with a thick atmosphere.",
			"Day 10.7 h; tilt 26.7° gives the rings their changing aspect from Earth.",
			"Orbital period 10,747 days, 29.4 years; dipole field 0.215 gauss-R³ aligned with the spin axis.",
		}, Source: nssdc},
	{Name: "Uranus", Kind: "planet", Color: "#9fd8e2", RadiusKm: 25559, Rings: true, Tilt: 97.8, Retro: true, DayHours: 17.2, PeriodD: 30589, Moons: 28, Surface: "uranus",
		Headline: "An ice giant rolling on its side, with the coldest atmosphere measured.",
		Facts: []string{
			"Diameter 51,118 km; mass 86.8 × 10^24 kg; gravity 8.7 m/s².",
			"Atmosphere 82.5% H₂, 15.2% He, 2.3% CH₄; methane absorbs red light, giving the cyan hue.",
			"-197 °C at 1 bar, and 53 K (-220 °C) at the 0.1-bar level.",
			"Axis tilted 97.8°, so each pole gets 42 years of sunlight then 42 of dark.",
			"Magnetic dipole tilted 58.6° from the spin axis; winds 0 to 250 m/s.",
			"28 known moons and a system of narrow dark rings; orbital period 30,589 days (84 years).",
		}, Source: nssdc},
	{Name: "Neptune", Kind: "planet", Color: "#4b6fe0", RadiusKm: 24764, Rings: true, Tilt: 28.3, DayHours: 16.1, PeriodD: 59800, Moons: 16, Surface: "neptune",
		Headline: "The windiest world, found by mathematics before telescopes in 1846.",
		Facts: []string{
			"Diameter 49,528 km; mass 102 × 10^24 kg; gravity 11.0 m/s².",
			"Atmosphere 80% H₂, 19% He, 1.5% CH₄; -201 °C at 1 bar.",
			"Winds up to 580 m/s, the fastest measured on any planet.",
			"Orbital period 59,800 days (164 years); it has completed one orbit since discovery.",
			"Magnetic dipole tilted 46.9°, offset from the planet's centre.",
			"16 known moons; Triton orbits backwards and is probably a captured Kuiper belt object.",
		}, Source: nssdc},

	// Dwarf planets and the big three asteroids: JPL SBDB elements.
	{Name: "Ceres", Kind: "dwarf", Color: "#a8a39b", RadiusKm: 469.7, DayHours: 9.07, PeriodD: 1680, Surface: "craters",
		small:    sb(2.765552595, 0.0796922951, 10.58802780, 80.24862682, 73.29421453, 274.4193464, 2461200.5, 1679.853120),
		Headline: "Largest object in the asteroid belt and the only dwarf planet inside Neptune.",
		Facts: []string{
			"Diameter 939 km; mass 9.39 × 10^20 kg, about a third of the whole belt.",
			"Density 2,162 kg/m³: a rocky body with a lot of water ice.",
			"Orbits at 2.77 au in 4.6 years; inclination 10.6°.",
			"Rotates in 9.07 h; albedo only 0.09.",
			"Discovered 1801 by Piazzi; visited by NASA's Dawn 2015 to 2018.",
		}, Source: sbdb},
	{Name: "Vesta", Kind: "asteroid", Color: "#c9c2b4", RadiusKm: 262.7, DayHours: 5.34, PeriodD: 1325, Surface: "craters",
		small:    sb(2.361365965, 0.0902037438, 7.143925545, 103.7012933, 151.4686478, 81.19015608, 2461200.5, 1325.389043),
		Headline: "Second most massive asteroid, bright enough to see with the naked eye.",
		Facts: []string{
			"Mean diameter 523 km (569 × 555 × 453 km); mass 2.59 × 10^20 kg.",
			"Density 3,460 kg/m³, a differentiated body with an iron core.",
			"Orbits at 2.36 au in 3.6 years; spins in 5.34 h.",
			"Albedo 0.42, the brightest large asteroid.",
			"Dawn orbited it 2011 to 2012.",
		}, Source: sbdb},
	{Name: "Pallas", Kind: "asteroid", Color: "#9a9790", RadiusKm: 256, DayHours: 7.81, PeriodD: 1684, Surface: "craters",
		small:    sb(2.769559011, 0.2307000996, 34.93279322, 172.8866193, 310.9699162, 254.2496522, 2461200.5, 1683.504810),
		Headline: "Third largest asteroid, on a steeply tilted orbit.",
		Facts: []string{
			"Mean diameter 513 km; mass 2.05 × 10^20 kg.",
			"Inclination 34.9°, the most tilted of the big asteroids.",
			"Eccentricity 0.231: 2.13 to 3.41 au from the Sun.",
			"Density 2,890 kg/m³; rotates in 7.81 h.",
		}, Source: sbdb},
	{Name: "Pluto", Kind: "dwarf", Color: "#d8c2a8", RadiusKm: 1188, Tilt: 119.5, Retro: true, DayHours: 153.29, PeriodD: 90560, Moons: 5, Surface: "pluto",
		small:    sb(39.58862939, 0.2518378779, 17.14771141, 110.2923841, 113.7090015, 38.68366347, 2457588.5, 90981.71648),
		Headline: "The first Kuiper belt object found, in 1930, with a nitrogen-ice heart.",
		Facts: []string{
			"Radius 1,188 km; mass 1.303 × 10^22 kg; gravity 0.62 m/s².",
			"Surface 24 to 38 K (-247 to -233 °C); pressure about 13 microbar.",
			"Atmosphere 99% N₂ with 0.5% CH₄ and 0.05% CO.",
			"Orbit e = 0.244, 29.6 to 49.6 au, inclined 17°; in a 3:2 resonance with Neptune.",
			"Day 153.3 h, retrograde; axis tilted 119.5°.",
			"Five moons; Charon (606 km radius) orbits 19,596 km away. New Horizons flew by 14 July 2015.",
		}, Source: nssdc + "; " + sbdb},
	{Name: "Haumea", Kind: "dwarf", Color: "#e9e4dc", RadiusKm: 780, DayHours: 3.92, PeriodD: 103208, Moons: 2, Surface: "craters",
		small:    sb(43.06029024, 0.1944430149, 28.20847393, 121.7860561, 240.6905473, 223.2104119, 2461200.5, 103208.1173),
		Headline: "An egg-shaped dwarf planet spinning once every 3.9 hours.",
		Facts: []string{
			"Orbits at 43.1 au (34.7 to 51.4 au), inclined 28.2°, period 283 years.",
			"Rotation 3.92 h, the fastest of any large body in the solar system.",
			"Two known moons and a thin ring discovered in 2017.",
		}, Source: sbdb},
	{Name: "Makemake", Kind: "dwarf", Color: "#d9a98a", RadiusKm: 715, DayHours: 22.8, PeriodD: 112365, Moons: 1, Surface: "craters",
		small:    sb(45.57093317, 0.1588889954, 29.02785604, 79.29483382, 297.0922733, 169.9379962, 2461200.5, 112364.8069),
		Headline: "A reddish Kuiper belt dwarf planet, discovered 2005.",
		Facts: []string{
			"Orbits at 45.6 au (38.3 to 52.8 au), inclined 29°, period 308 years.",
			"Rotation 22.8 h.",
			"One known moon.",
		}, Source: sbdb},
	{Name: "Eris", Kind: "dwarf", Color: "#efefef", RadiusKm: 1163, DayHours: 25.9, PeriodD: 204517, Moons: 1, Surface: "craters",
		small:    sb(67.93394688, 0.4382385348, 43.92582795, 36.00477044, 150.7949236, 211.7744343, 2461200.5, 204516.6629),
		Headline: "The scattered-disk dwarf planet that demoted Pluto.",
		Facts: []string{
			"Orbits at 67.9 au, swinging from 38.2 to 97.7 au; inclined 43.9°.",
			"Period 560 years; currently near aphelion, so it moves slowly.",
			"Slightly more massive than Pluto; one moon, Dysnomia.",
		}, Source: sbdb},

	// Comets.
	{Name: "Halley", Kind: "comet", Color: "#cfe6ff", RadiusKm: 5.5, PeriodD: 27728, Surface: "comet",
		small:    sb(17.92863505, 0.9679359957, 162.1905300, 59.09894721, 112.2414315, 274.3823371, 2439875.5, 27728.04609),
		Headline: "1P/Halley: the first comet shown to return, every 75 to 76 years.",
		Facts: []string{
			"Nucleus 15 × 8 km, albedo 0.04, darker than coal.",
			"Perihelion 0.575 au, aphelion 35.3 au, beyond Neptune.",
			"Inclination 162°: it orbits backwards relative to the planets.",
			"Last perihelion 9 February 1986; next expected 2061.",
			"Source of the Eta Aquariid and Orionid meteor showers.",
		}, Source: sbdb},
	{Name: "Encke", Kind: "comet", Color: "#cfe6ff", RadiusKm: 2.4, PeriodD: 1207, Surface: "comet",
		small:    sb(2.218624764, 0.8469404309, 11.33675915, 334.0171400, 187.2881999, 36.82518175, 2460363.5, 1207.047084),
		Headline: "2P/Encke: the shortest-period comet known, 3.3 years.",
		Facts: []string{
			"Nucleus 4.8 km; rotation 11.1 h.",
			"Perihelion 0.34 au, inside Mercury's orbit; aphelion 4.1 au.",
			"Parent of the Taurid meteor streams.",
		}, Source: sbdb},
	{Name: "67P", Kind: "comet", Color: "#cfe6ff", RadiusKm: 1.7, PeriodD: 2353, Surface: "comet",
		small:    sb(3.462249490, 0.6409081309, 7.040294938, 50.13557377, 12.79824970, 8.859927425, 2457305.5, 2353.076068),
		Headline: "67P/Churyumov-Gerasimenko: the rubber-duck comet Rosetta orbited.",
		Facts: []string{
			"Nucleus 3.4 km, two lobes; density only 533 kg/m³.",
			"Perihelion 1.24 au, aphelion 5.68 au near Jupiter; period 6.4 years.",
			"ESA's Rosetta orbited it 2014 to 2016 and landed Philae on it.",
		}, Source: sbdb},
	{Name: "Hale-Bopp", Kind: "comet", Color: "#cfe6ff", RadiusKm: 30, PeriodD: 863280, Surface: "comet",
		small:    sb(177.4333839, 0.9949810028, 89.28759425, 282.7334214, 130.4146671, 3.878386339, 2459837.5, 863279.5035),
		Headline: "C/1995 O1 Hale-Bopp: the great comet of 1997, from the Oort cloud.",
		Facts: []string{
			"Nucleus about 60 km, unusually large.",
			"Perihelion 0.89 au on 1 April 1997; aphelion about 354 au.",
			"Orbit tipped 89°, almost perpendicular to the planets; period roughly 2,400 years.",
			"Visible to the naked eye for a record 18 months.",
		}, Source: sbdb},
}

func init() {
	for _, b := range bodies {
		if p, ok := planetTable[b.Name]; ok {
			pp := p
			b.planet = &pp
		}
	}
}

// Belt is a statistical population of small bodies, each with its own
// element set, generated from a fixed seed so every run looks the same.
type Belt struct {
	Name  string
	Elems []Elements
}

var belts []Belt

// Kirkwood gaps: mean-motion resonances with Jupiter clear these distances.
var kirkwood = []struct{ a, half float64 }{
	{2.502, 0.03},  // 3:1
	{2.825, 0.02},  // 5:2
	{2.958, 0.015}, // 7:3
	{3.279, 0.04},  // 2:1
}

func init() {
	rng := rand.New(rand.NewSource(42))
	rayleigh := func(sigma, max float64) float64 {
		for {
			v := sigma * math.Sqrt(-2*math.Log(1-rng.Float64()))
			if v < max {
				return v
			}
		}
	}
	// Main belt: 2.1 to 3.3 au, denser toward the middle, with gaps.
	main := Belt{Name: "main"}
	for len(main.Elems) < 1400 {
		a := 2.1 + 1.2*math.Pow(rng.Float64(), 0.9)
		gap := false
		for _, k := range kirkwood {
			if math.Abs(a-k.a) < k.half {
				gap = true
			}
		}
		if gap {
			continue
		}
		main.Elems = append(main.Elems, Elements{A: a, E: rayleigh(0.1, 0.35), I: rayleigh(7, 30),
			Om: rng.Float64() * 360, W: rng.Float64() * 360, M0: rng.Float64() * 360,
			Epoch: j2000, N: meanMotion(a)})
	}
	// Jupiter Trojans: share Jupiter's orbit, 60° ahead and behind.
	jup := planetTable["Jupiter"]
	tro := Belt{Name: "trojans"}
	for i := 0; i < 500; i++ {
		lead := 60.0
		if i%2 == 1 {
			lead = -60
		}
		libr := rng.NormFloat64() * 12
		a := jup.A + rng.NormFloat64()*0.12
		// Mean longitude follows Jupiter's with an offset; use L - wbar for M.
		L := jup.L + lead + libr
		tro.Elems = append(tro.Elems, Elements{A: a, E: rayleigh(0.07, 0.25), I: rayleigh(10, 35),
			Om: jup.Om, W: jup.Wbar - jup.Om, M0: L - jup.Wbar, Epoch: j2000, N: jup.DL / 36525})
	}
	// Kuiper belt: classical belt 42 to 48 au plus plutinos clustered at 39.4 au.
	kb := Belt{Name: "kuiper"}
	for i := 0; i < 1600; i++ {
		var a float64
		if i%4 == 0 {
			a = 39.4 + rng.NormFloat64()*0.3
		} else {
			a = 42 + 6*rng.Float64()
		}
		kb.Elems = append(kb.Elems, Elements{A: a, E: rayleigh(0.08, 0.3), I: rayleigh(6, 30),
			Om: rng.Float64() * 360, W: rng.Float64() * 360, M0: rng.Float64() * 360,
			Epoch: j2000, N: meanMotion(a)})
	}
	belts = []Belt{main, tro, kb}
}
