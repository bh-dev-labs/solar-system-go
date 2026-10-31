# Solar system in Go

An educational, real-ephemeris model of the solar system. Go does the physics
and the texture generation; the browser only draws.

## Run

```bash
cd solar-system-go
go build -o solarsystem.exe .
./solarsystem.exe            # http://127.0.0.1:8080
```

Standard library only, no modules to download. Tests: `go test ./...`.

## What it shows

- Sun, eight planets, Ceres, Vesta, Pallas, Pluto, Haumea, Makemake, Eris.
- Comets 1P/Halley, 2P/Encke, 67P/Churyumov-Gerasimenko and C/1995 O1
  Hale-Bopp on their real inclined, eccentric orbits. Inside 6 au a comet grows
  a straight blue ion tail pointing away from the Sun and a broader, curved
  dust tail that lags along the orbit.
- Main asteroid belt (1,400 bodies, 2.1 to 3.3 au) with the Kirkwood gaps at
  the 3:1, 5:2, 7:3 and 2:1 Jupiter resonances left empty; 500 Jupiter Trojans
  librating around L4 and L5; a Kuiper belt of 1,600 bodies with a plutino
  cluster at 39.4 au.
- Click any body for a fact card: live distance, speed, light time, Earth
  distance, plus sourced facts and a procedural rotating surface.

## Controls

Time warp from paused to 100 years per second; date picker (1800 to 2050, the
validity range of the JPL element table); Today and J2000; tilt slider to see
orbital inclinations; schematic or true-scale distances; zoom with the wheel,
drag to pan, focus on any body. The canvas renders at native device pixel
ratio and resizes with the window, so it fills a 4K monitor as well as a
laptop.

## How the physics works

`orbit.go` implements the JPL "Approximate Positions of the Planets" method:
Keplerian elements at J2000 plus per-century rates, mean anomaly from
`L - ϖ`, Kepler's equation `M = E - e sin E` solved by Newton's method,
perifocal coordinates rotated by `-ω, -I, -Ω` into heliocentric J2000
ecliptic coordinates. Small bodies use fixed osculating elements from the JPL
Small-Body Database with mean motion `360 / period`. Speeds come from the
vis-viva equation.

Known simplifications: no planetary perturbations, no TT-UTC offset (about
69 s), Earth is the Earth-Moon barycentre. Fine for visualisation, not for
pointing a telescope.

## Sources

- Planet and Pluto physical data: NASA NSSDC Planetary Fact Sheet, updated
  18 March 2025, and the per-planet fact sheets (atmospheres, pressures, winds,
  magnetic fields). https://nssdc.gsfc.nasa.gov/planetary/factsheet/
- Sun: NSSDC Sun Fact Sheet.
- Planet orbital elements: JPL SSD, Keplerian elements for approximate
  positions, table 1 (1800 to 2050). https://ssd.jpl.nasa.gov/planets/approx_pos.html
- Asteroid, dwarf planet and comet elements and sizes: JPL Small-Body Database
  API, fetched 19 October 2026. https://ssd-api.jpl.nasa.gov/sbdb.api
- Belt mass, Ceres, Pallas and Vesta sizes: NSSDC Asteroid Fact Sheet.
- Kuiper belt extent and population: NASA Science, Kuiper Belt facts.
- Kirkwood gap positions (2.502, 2.825, 2.958, 3.279 au) are the standard
  resonance locations and are illustrative; the belt itself is a seeded
  statistical population, not a catalogue.

Surface textures are procedural illustrations generated in `surface.go`
(bands, a Great Red Spot, a Neptune dark spot, Pluto's bright Sputnik Planitia
heart, cratered terrain), not photographs.
