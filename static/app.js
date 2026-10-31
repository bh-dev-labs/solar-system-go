"use strict";
// Drawing only. Positions, speeds and textures come from the Go server.

const canvas = document.getElementById("sky");
const ctx = canvas.getContext("2d");
const $ = (id) => document.getElementById(id);

const SPEEDS = [0, 1 / 24, 1, 7, 30, 365.25, 3652.5, 36525]; // days per second
const SPEED_LABELS = ["paused", "1 h/s", "1 day/s", "1 week/s", "1 month/s", "1 year/s", "10 years/s", "100 years/s"];
const JD_MIN = 2378497.5, JD_MAX = 2470172.5; // 1800-01-01 to 2050-12-31, JPL table validity

const view = {
  jd: Date.now() / 86400000 + 2440587.5,
  speedIdx: 3, paused: false, schematic: true, tilt: 60, zoom: 0,
  panX: 0, panY: 0, dragX: 0, dragY: 0, focus: "", selected: null,
  layers: { orbits: true, labels: true, main: true, trojans: true, kuiper: true, comets: true, dwarfs: true },
};

let meta = [], metaByName = {}, state = null, orbits = {}, stars = [];
let dpr = 1, W = 0, H = 0;

// Canvas fills the window at native resolution on any monitor.
function resize() {
  dpr = Math.min(window.devicePixelRatio || 1, 3);
  W = window.innerWidth; H = window.innerHeight;
  canvas.width = Math.round(W * dpr); canvas.height = Math.round(H * dpr);
  canvas.style.width = W + "px"; canvas.style.height = H + "px";
  ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
  stars = [];
  const n = Math.round((W * H) / 900);
  let s = 1234567;
  const rnd = () => (s = (s * 16807) % 2147483647) / 2147483647;
  for (let i = 0; i < n; i++) stars.push([rnd() * W, rnd() * H, 0.3 + rnd() * 1.1, 0.3 + rnd() * 0.7]);
}
window.addEventListener("resize", resize);
resize();

// Radial mapping. Schematic compresses distance so Mercury and Neptune both fit;
// true scale is linear. Both are scaled by the smaller window dimension.
function unit() { return Math.min(W, H) * 0.48 * Math.pow(2, view.zoom); }
function radial(r) {
  return view.schematic ? Math.log1p(r / 0.25) / Math.log1p(32 / 0.25) : r / 32;
}
// On narrow windows the HUD panels cover part of the sky, so shift the
// centre into the free space between them. On big monitors shift is zero.
function centreShift() {
  if (W >= 1500) return 0;
  const left = $("controls").offsetWidth + 16;
  const right = $("info").classList.contains("show") ? $("info").offsetWidth + 16 : 0;
  return (left - right) / 2;
}
function project(x, y, z) {
  const r = Math.hypot(x, y, z);
  if (r === 0) return [W / 2 + centreShift() + view.panX, H / 2 + view.panY];
  const k = radial(r) / r * unit();
  const t = view.tilt * Math.PI / 180;
  const px = x * k, py = (y * Math.cos(t) - z * Math.sin(t)) * k;
  return [W / 2 + centreShift() + view.panX + px, H / 2 + view.panY - py];
}

function bodyPixelRadius(b) {
  if (!view.schematic) {
    return Math.max(1.2, (b.radiusKm / 149597870.7) / 32 * unit());
  }
  if (b.kind === "star") return 11 * Math.pow(2, view.zoom * 0.3);
  return Math.max(1.6, 1.15 * Math.pow(b.radiusKm / 1000, 0.42)) * Math.pow(2, view.zoom * 0.3);
}

const beltStyle = { main: ["#8a877f", 0.9], trojans: ["#b08a5a", 0.9], kuiper: ["#6f8fb3", 0.9] };

function draw() {
  ctx.fillStyle = "#04060c"; ctx.fillRect(0, 0, W, H);
  ctx.fillStyle = "#ffffff";
  for (const [x, y, r, a] of stars) { ctx.globalAlpha = a; ctx.beginPath(); ctx.arc(x, y, r, 0, 7); ctx.fill(); }
  ctx.globalAlpha = 1;
  if (!state) return;

  // Focus: pan so the chosen body sits at the centre.
  // Focus: keep the chosen body at the centre of the free sky, plus any
  // offset the user dragged in.
  if (view.focus) {
    const f = state.bodies.find((b) => b.n === view.focus);
    if (f) {
      view.panX = 0; view.panY = 0;
      const [fx, fy] = project(f.x, f.y, f.z);
      view.panX = W / 2 + centreShift() - fx + view.dragX; view.panY = H / 2 - fy + view.dragY;
    }
  }

  // Belts.
  for (const name of ["main", "trojans", "kuiper"]) {
    if (!view.layers[name] || !state.belts || !state.belts[name]) continue;
    const pts = state.belts[name]; const [col, sz] = beltStyle[name];
    ctx.fillStyle = col; ctx.globalAlpha = 0.75;
    for (let i = 0; i < pts.length; i += 3) {
      const [px, py] = project(pts[i], pts[i + 1], pts[i + 2]);
      if (px < -2 || py < -2 || px > W + 2 || py > H + 2) continue;
      ctx.fillRect(px, py, sz, sz);
    }
    ctx.globalAlpha = 1;
  }

  // Orbits.
  if (view.layers.orbits) {
    ctx.lineWidth = 0.8;
    for (const b of meta) {
      if (!visible(b) || !orbits[b.name]) continue;
      ctx.strokeStyle = b.kind === "comet" ? "rgba(207,230,255,0.28)" : b.kind === "dwarf" || b.kind === "asteroid" ? "rgba(216,194,168,0.22)" : "rgba(255,255,255,0.18)";
      ctx.beginPath();
      orbits[b.name].forEach((p, i) => { const [px, py] = project(p[0], p[1], p[2]); i ? ctx.lineTo(px, py) : ctx.moveTo(px, py); });
      ctx.stroke();
    }
  }

  // Bodies, far to near so nearer ones draw on top when tilted.
  const items = state.bodies.map((s) => ({ s, b: metaByName[s.n] })).filter((it) => it.b && visible(it.b));
  items.sort((p, q) => (q.s.y) - (p.s.y));
  const labelBoxes = [];
  for (const { s, b } of items) {
    const [px, py] = project(s.x, s.y, s.z);
    const rad = bodyPixelRadius(b);
    if (b.kind === "comet") drawComet(s, px, py, rad); else drawBody(b, s, px, py, rad);
    if (view.layers.labels) {
      ctx.font = `${view.selected === b.name ? 500 : 400} 12px system-ui, sans-serif`;
      ctx.fillStyle = view.selected === b.name ? "#ffffff" : "rgba(232,236,244,0.75)";
      const tw = ctx.measureText(b.name).width;
      // Try right, left, below, above; skip the label if every slot collides.
      const slots = [[px + rad + 5, py + 4], [px - rad - 5 - tw, py + 4], [px - tw / 2, py + rad + 14], [px - tw / 2, py - rad - 6]];
      for (const [lx, ly] of slots) {
        const box = [lx - 2, ly - 12, lx + tw + 2, ly + 3];
        if (!labelBoxes.some((o) => box[0] < o[2] && box[2] > o[0] && box[1] < o[3] && box[3] > o[1])) {
          labelBoxes.push(box); ctx.fillText(b.name, lx, ly); break;
        }
      }
    }
    if (view.selected === b.name) {
      ctx.strokeStyle = "#7fb3ff"; ctx.lineWidth = 1; ctx.beginPath(); ctx.arc(px, py, rad + 6, 0, 7); ctx.stroke();
    }
    s._px = px; s._py = py; s._rad = rad;
  }
}

function visible(b) {
  if (b.kind === "comet") return view.layers.comets;
  if (b.kind === "dwarf" || b.kind === "asteroid") return view.layers.dwarfs;
  return true;
}

function drawBody(b, s, px, py, rad) {
  if (b.kind === "star") {
    const g = ctx.createRadialGradient(px, py, 0, px, py, rad * 3.2);
    g.addColorStop(0, "rgba(255,230,160,0.55)"); g.addColorStop(1, "rgba(255,210,120,0)");
    ctx.fillStyle = g; ctx.beginPath(); ctx.arc(px, py, rad * 3.2, 0, 7); ctx.fill();
    ctx.fillStyle = b.color; ctx.beginPath(); ctx.arc(px, py, rad, 0, 7); ctx.fill();
    return;
  }
  if (b.rings && rad > 2.5) {
    const rr = b.name === "Saturn" ? 2.3 : 1.6;
    ctx.strokeStyle = b.name === "Saturn" ? "rgba(227,207,156,0.85)" : "rgba(200,210,230,0.35)";
    ctx.lineWidth = b.name === "Saturn" ? rad * 0.55 : rad * 0.15;
    ctx.beginPath(); ctx.ellipse(px, py, rad * rr, rad * rr * 0.38, -0.35, 0, 7); ctx.stroke();
  }
  // Lit side faces the Sun.
  const [sx, sy] = project(0, 0, 0);
  const ang = Math.atan2(sy - py, sx - px);
  const g = ctx.createRadialGradient(px + Math.cos(ang) * rad * 0.5, py + Math.sin(ang) * rad * 0.5, rad * 0.1, px, py, rad);
  g.addColorStop(0, lighten(b.color, 0.35)); g.addColorStop(0.7, b.color); g.addColorStop(1, darken(b.color, 0.55));
  ctx.fillStyle = g; ctx.beginPath(); ctx.arc(px, py, rad, 0, 7); ctx.fill();
}

function drawComet(s, px, py, rad) {
  const [sx, sy] = project(0, 0, 0);
  const ang = Math.atan2(py - sy, px - sx); // anti-sunward
  const r = s.r;
  if (r < 6) {
    // Ion tail: straight, anti-sunward. Dust tail: broader, lags along the orbit.
    const len = Math.min(220, 28 / (r * r) * Math.pow(2, view.zoom * 0.5) + 10);
    const grad = (a, w, col) => {
      const ex = px + Math.cos(a) * len, ey = py + Math.sin(a) * len;
      const g = ctx.createLinearGradient(px, py, ex, ey);
      g.addColorStop(0, col); g.addColorStop(1, "rgba(0,0,0,0)");
      ctx.strokeStyle = g; ctx.lineWidth = w; ctx.lineCap = "round";
      ctx.beginPath(); ctx.moveTo(px, py); ctx.lineTo(ex, ey); ctx.stroke();
    };
    grad(ang, 2, "rgba(130,180,255,0.75)");
    grad(ang + 0.28, 5, "rgba(255,240,210,0.35)");
    const g = ctx.createRadialGradient(px, py, 0, px, py, rad + 6);
    g.addColorStop(0, "rgba(220,235,255,0.8)"); g.addColorStop(1, "rgba(220,235,255,0)");
    ctx.fillStyle = g; ctx.beginPath(); ctx.arc(px, py, rad + 6, 0, 7); ctx.fill();
  }
  ctx.fillStyle = "#e8f0ff"; ctx.beginPath(); ctx.arc(px, py, Math.max(1.5, rad * 0.8), 0, 7); ctx.fill();
}

function hex(c) { return [1, 3, 5].map((i) => parseInt(c.slice(i, i + 2), 16)); }
function lighten(c, t) { const [r, g, b] = hex(c); return `rgb(${r + (255 - r) * t | 0},${g + (255 - g) * t | 0},${b + (255 - b) * t | 0})`; }
function darken(c, t) { const [r, g, b] = hex(c); return `rgb(${r * (1 - t) | 0},${g * (1 - t) | 0},${b * (1 - t) | 0})`; }

// Networking: ask Go for the state at the current JD, about 30 times a second.
let fetching = false, lastTick = performance.now();
async function tick() {
  const now = performance.now(); const dt = (now - lastTick) / 1000; lastTick = now;
  if (!view.paused) view.jd += SPEEDS[view.speedIdx] * dt;
  if (view.jd < JD_MIN) view.jd = JD_MIN; if (view.jd > JD_MAX) view.jd = JD_MAX;
  if (!fetching) {
    fetching = true;
    try {
      const r = await fetch(`/api/state?jd=${view.jd}`);
      state = await r.json();
      $("date").textContent = state.date;
      $("pick").value = state.date.slice(0, 10);
      if (view.selected) updateLive();
    } catch (e) { /* server away; keep last frame */ }
    fetching = false;
  }
}
setInterval(tick, 33);
(function frame() { draw(); requestAnimationFrame(frame); })();

async function loadOrbits() {
  orbits = await (await fetch(`/api/orbits?jd=${view.jd}`)).json();
}

async function init() {
  meta = await (await fetch("/api/bodies")).json();
  for (const b of meta) metaByName[b.name] = b;
  const sel = $("focus");
  for (const b of meta) if (b.kind !== "star") { const o = document.createElement("option"); o.value = b.name; o.textContent = b.name; sel.appendChild(o); }
  await loadOrbits();
  setSpeed(view.speedIdx);
}
init();

// Fact panel.
function select(name) {
  view.selected = name;
  const b = metaByName[name];
  if (!b) { $("info").classList.remove("show"); return; }
  $("kind").textContent = b.kind === "dwarf" ? "dwarf planet" : b.kind;
  $("name").textContent = b.name;
  $("headline").textContent = b.headline;
  const globe = $("globe");
  globe.style.backgroundImage = `url(/surface/${b.surface}.png)`;
  globe.classList.toggle("retro", !!b.retrograde);
  globe.style.animationDuration = b.dayHours ? `${Math.max(4, Math.min(60, 4 * Math.log2(1 + b.dayHours)))}s` : "24s";
  $("facts").innerHTML = "";
  for (const f of b.facts) { const li = document.createElement("li"); li.textContent = f; $("facts").appendChild(li); }
  $("src").textContent = "Source: " + b.source + ". Surface texture is procedural, generated by the Go server, not a photograph.";
  updateLive();
  $("info").classList.add("show");
}

function updateLive() {
  const b = metaByName[view.selected]; const s = state && state.bodies.find((x) => x.n === view.selected);
  const live = $("live"); live.innerHTML = "";
  const add = (k, v) => { const sp = document.createElement("span"); sp.textContent = k; const bb = document.createElement("b"); bb.textContent = v; live.append(sp, bb); };
  if (!b || !s) return;
  if (b.kind === "star") { add("Rotation", "25.4 days"); return; }
  add("Distance from Sun", `${s.r.toFixed(3)} au`);
  add("Orbital speed", `${s.v.toFixed(2)} km/s`);
  add("Light time from Sun", s.lt < 90 ? `${s.lt.toFixed(1)} min` : `${(s.lt / 60).toFixed(1)} h`);
  const e = state.bodies.find((x) => x.n === "Earth");
  if (e && b.name !== "Earth") add("Distance from Earth", `${Math.hypot(s.x - e.x, s.y - e.y, s.z - e.z).toFixed(3)} au`);
  if (b.periodDays) add("Orbital period", b.periodDays > 1000 ? `${(b.periodDays / 365.25).toFixed(1)} years` : `${b.periodDays} days`);
  if (b.dayHours) add("Day length", b.dayHours > 72 ? `${(b.dayHours / 24).toFixed(1)} Earth days` : `${b.dayHours} h`);
  if (b.tiltDeg) add("Axial tilt", `${b.tiltDeg}°`);
  if (b.moons) add("Known moons", String(b.moons));
}

// Input.
let dragging = null;
canvas.addEventListener("pointerdown", (e) => {
  dragging = { x: e.clientX, y: e.clientY, px: view.panX, py: view.panY, dx: view.dragX, dy: view.dragY, moved: false };
});
canvas.addEventListener("pointermove", (e) => {
  if (!dragging) return;
  const dx = e.clientX - dragging.x, dy = e.clientY - dragging.y;
  if (Math.hypot(dx, dy) > 3) dragging.moved = true;
  if (!dragging.moved) return;
  if (view.focus) { view.dragX = dragging.dx + dx; view.dragY = dragging.dy + dy; }
  else { view.panX = dragging.px + dx; view.panY = dragging.py + dy; }
});
canvas.addEventListener("pointerup", (e) => {
  if (dragging && !dragging.moved && state) {
    let best = null, bd = 14;
    for (const s of state.bodies) {
      if (s._px === undefined) continue;
      const d = Math.hypot(e.clientX - s._px, e.clientY - s._py) - Math.min(s._rad, 20);
      if (d < bd) { bd = d; best = s.n; }
    }
    if (best) select(best); else { view.selected = null; $("info").classList.remove("show"); }
  }
  dragging = null;
});
canvas.addEventListener("wheel", (e) => {
  e.preventDefault();
  view.zoom = Math.max(-2, Math.min(3, view.zoom - e.deltaY * 0.0015));
  $("zoom").value = view.zoom; $("zoomv").textContent = Math.pow(2, view.zoom).toFixed(2) + "×";
}, { passive: false });

function setSpeed(i) { view.speedIdx = i; $("speed").value = i; $("speedv").textContent = SPEED_LABELS[i]; }
$("speed").oninput = (e) => setSpeed(+e.target.value);
$("pause").onclick = () => { view.paused = !view.paused; $("pause").firstChild.textContent = view.paused ? "Play" : "Pause"; $("pause").classList.toggle("on", view.paused); };
$("today").onclick = () => { view.jd = Date.now() / 86400000 + 2440587.5; loadOrbits(); };
$("j2000").onclick = () => { view.jd = 2451545.0; loadOrbits(); };
$("pick").onchange = (e) => { const t = Date.parse(e.target.value + "T12:00:00Z"); if (!isNaN(t)) { view.jd = t / 86400000 + 2440587.5; loadOrbits(); } };
$("scale").onclick = () => { view.schematic = !view.schematic; $("scale").textContent = view.schematic ? "Schematic" : "True scale"; $("scale").classList.toggle("on", view.schematic); };
$("tilt").oninput = (e) => { view.tilt = +e.target.value; $("tiltbtn").textContent = `Tilt ${view.tilt}°`; };
$("tiltbtn").onclick = () => { view.tilt = view.tilt ? 0 : 60; $("tilt").value = view.tilt; $("tiltbtn").textContent = `Tilt ${view.tilt}°`; };
$("zoom").oninput = (e) => { view.zoom = +e.target.value; $("zoomv").textContent = Math.pow(2, view.zoom).toFixed(2) + "×"; };
$("focus").onchange = (e) => { view.focus = e.target.value; view.dragX = 0; view.dragY = 0; if (!view.focus) { view.panX = 0; view.panY = 0; } };
$("close").onclick = () => { view.selected = null; $("info").classList.remove("show"); };
document.querySelectorAll("[data-layer]").forEach((btn) => btn.onclick = () => { const k = btn.dataset.layer; view.layers[k] = !view.layers[k]; btn.classList.toggle("on", view.layers[k]); });
window.addEventListener("keydown", (e) => { if (e.key === " ") { e.preventDefault(); $("pause").click(); } });
