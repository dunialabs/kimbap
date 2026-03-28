#!/usr/bin/env python3
"""
Generate kimbap logo.svg and favicon.ico from the splash.go algorithm.

Faithfully reproduces the procedural kimbap cross-section art defined in
internal/splash/splash.go using the exact same color palette and cell logic.
"""

import math
import os
import sys
from PIL import Image


# ── Colors (exact values from splash.go) ───────────────────────────

COLORS = {
    "nori1":  (26,  74,  26),
    "nori2":  (46,  110, 46),
    "rice1":  (250, 248, 240),
    "rice2":  (240, 234, 216),
    "sesame": (200, 160, 48),
    "green1": (52,  160, 52),
    "green2": (42,  122, 42),
    "orange": (240, 120, 32),
    "yolk1":  (244, 208, 48),
    "yolk2":  (248, 240, 64),
    "white":  (253, 244, 240),
    "red1":   (232, 72,  72),
    "red2":   (224, 64,  64),
}

RX = 5.6
RY = 4.8


# ── Algorithm (exact port of getCell from splash.go) ───────────────

def get_cell(nx, ny):
    """Return (r, g, b) for the normalized position, or None if outside."""
    px = nx * RX
    py = ny * RY
    d = math.sqrt(nx * nx + ny * ny)

    if d > 1.00:
        return None
    if d > 0.92:
        return COLORS["nori1"]
    if d > 0.84:
        return COLORS["nori2"]

    # Rice area
    if d > 0.52:
        speckle = abs(math.sin(px * 7.3 + py * 3.1)) > 0.92
        if speckle and d < 0.68:
            return COLORS["sesame"]
        if d > 0.70:
            return COLORS["rice1"]
        return COLORS["rice2"]

    # Inner fillings (switch-case order matters — first match wins)
    if py < -1.5 and abs(px) < 1.8:
        return COLORS["green1"]
    if py < -0.8 and abs(px) < 2.4 and abs(px) > 1.4:
        return COLORS["green2"]
    if px < -1.0 and px > -2.4 and py > -1.5 and py < 0.3:
        return COLORS["orange"]
    if py > -1.5 and py < -0.3 and abs(px) < 1.0:
        return COLORS["yolk1"]
    if py > -0.4 and py < 0.8 and abs(px) < 1.2:
        return COLORS["yolk2"]
    if px > 0.9 and px < 2.4 and py > -1.5 and py < 0.9:
        if int(math.floor((py + 1.5) * 1.6)) % 2 == 0:
            return COLORS["white"]
        return COLORS["red1"]
    if py > 0.7 and py < 2.0 and abs(px) < 1.7:
        return COLORS["red2"]

    return COLORS["rice2"]


# ── Bitmap rendering ───────────────────────────────────────────────

def render_bitmap(size, padding=0.90):
    """Render the kimbap algorithm to an RGBA bitmap."""
    img = Image.new("RGBA", (size, size), (0, 0, 0, 0))
    pixels = img.load()

    center = size / 2.0
    scale = center * padding

    for y in range(size):
        for x in range(size):
            nx = (x + 0.5 - center) / scale
            ny = (y + 0.5 - center) / scale
            color = get_cell(nx, ny)
            if color:
                pixels[x, y] = (*color, 255)

    return img


def render_bitmap_aa(size, supersample=4, padding=0.90):
    """Render with supersampled anti-aliasing for smooth edges."""
    big = render_bitmap(size * supersample, padding)
    return big.resize((size, size), Image.LANCZOS)


# ── ICO generation ─────────────────────────────────────────────────

def generate_ico(path, sizes=(16, 32, 48)):
    """Generate a multi-size .ico file."""
    master = render_bitmap_aa(256, supersample=4, padding=0.88)
    master.save(path, format="ICO", sizes=[(s, s) for s in sizes])
    print(f"  ICO → {path}  ({', '.join(f'{s}×{s}' for s in sizes)})")


# ── SVG generation ─────────────────────────────────────────────────

def _hex(name):
    r, g, b = COLORS[name]
    return f"#{r:02x}{g:02x}{b:02x}"


def generate_svg(path):
    """Generate a clean vector SVG of the kimbap cross-section."""
    S = 512
    C = S / 2
    R = S / 2 * 0.92

    r_outer    = R           # d = 1.00
    r_nori_mid = R * 0.92    # d = 0.92
    r_nori_in  = R * 0.84    # d = 0.84
    r_rice_mid = R * 0.70    # d = 0.70
    r_inner    = R * 0.52    # d = 0.52
    def sx(px_val):
        return C + (px_val / RX) * R

    def sy_val(py_val):
        return C + (py_val / RY) * R

    # ── Sesame seeds ──
    raw_seeds = []
    for a in range(0, 360, 3):
        for d_100 in range(53, 68):
            d = d_100 / 100.0
            nx = d * math.cos(math.radians(a))
            ny = d * math.sin(math.radians(a))
            px, py = nx * RX, ny * RY
            if abs(math.sin(px * 7.3 + py * 3.1)) > 0.92:
                raw_seeds.append((C + nx * R, C + ny * R))

    seeds = []
    for x, y in raw_seeds:
        if not any((x - kx) ** 2 + (y - ky) ** 2 < 144 for kx, ky in seeds):
            seeds.append((x, y))

    # ── Crab-stick bands (alternating white / red) ──
    bands = []
    n = 0
    while True:
        py_lo = -1.5 + n / 1.6
        if py_lo >= 0.9:
            break
        py_hi = min(-1.5 + (n + 1) / 1.6, 0.9)
        color = _hex("white") if n % 2 == 0 else _hex("red1")
        bands.append(
            f'    <rect x="{sx(0.9):.1f}" y="{sy_val(py_lo):.1f}" '
            f'width="{sx(2.4) - sx(0.9):.1f}" height="{sy_val(py_hi) - sy_val(py_lo):.1f}" '
            f'fill="{color}" rx="2"/>'
        )
        n += 1

    # ── Assemble SVG ──
    svg_parts = [
        f'<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 {S} {S}">',
        "  <defs>",
        f'    <clipPath id="fc">',
        f'      <circle cx="{C}" cy="{C}" r="{r_inner:.1f}"/>',
        "    </clipPath>",
        "  </defs>",
        "",
        "  <!-- Nori (seaweed wrap) -->",
        f'  <circle cx="{C}" cy="{C}" r="{r_outer:.1f}" fill="{_hex("nori1")}"/>',
        f'  <circle cx="{C}" cy="{C}" r="{r_nori_mid:.1f}" fill="{_hex("nori2")}"/>',
        "",
        "  <!-- Rice -->",
        f'  <circle cx="{C}" cy="{C}" r="{r_nori_in:.1f}" fill="{_hex("rice1")}"/>',
        f'  <circle cx="{C}" cy="{C}" r="{r_rice_mid:.1f}" fill="{_hex("rice2")}"/>',
        "",
        "  <!-- Inner filling base -->",
        f'  <circle cx="{C}" cy="{C}" r="{r_inner:.1f}" fill="{_hex("rice2")}"/>',
        "",
        "  <!-- Fillings (drawn bottom→top so higher-priority covers lower) -->",
        '  <g clip-path="url(#fc)">',
        "",
        "    <!-- Pickled radish / red (bottom) -->",
        f'    <rect x="{sx(-1.7):.1f}" y="{sy_val(0.7):.1f}" '
        f'width="{sx(1.7) - sx(-1.7):.1f}" height="{sy_val(2.0) - sy_val(0.7):.1f}" '
        f'fill="{_hex("red2")}" rx="3"/>',
        "",
        "    <!-- Crab stick (right) -->",
        *bands,
        "",
        "    <!-- Egg – outer yolk -->",
        f'    <rect x="{sx(-1.2):.1f}" y="{sy_val(-0.4):.1f}" '
        f'width="{sx(1.2) - sx(-1.2):.1f}" height="{sy_val(0.8) - sy_val(-0.4):.1f}" '
        f'fill="{_hex("yolk2")}" rx="3"/>',
        "",
        "    <!-- Egg – inner yolk -->",
        f'    <rect x="{sx(-1.0):.1f}" y="{sy_val(-1.5):.1f}" '
        f'width="{sx(1.0) - sx(-1.0):.1f}" height="{sy_val(-0.3) - sy_val(-1.5):.1f}" '
        f'fill="{_hex("yolk1")}" rx="3"/>',
        "",
        "    <!-- Carrot (left) -->",
        f'    <rect x="{sx(-2.4):.1f}" y="{sy_val(-1.5):.1f}" '
        f'width="{sx(-1.0) - sx(-2.4):.1f}" height="{sy_val(0.3) - sy_val(-1.5):.1f}" '
        f'fill="{_hex("orange")}" rx="3"/>',
        "",
        "    <!-- Spinach / cucumber (sides) -->",
        f'    <rect x="{sx(-2.4):.1f}" y="{sy_val(-5.0):.1f}" '
        f'width="{sx(-1.4) - sx(-2.4):.1f}" height="{sy_val(-0.8) - sy_val(-5.0):.1f}" '
        f'fill="{_hex("green2")}" rx="2"/>',
        f'    <rect x="{sx(1.4):.1f}" y="{sy_val(-5.0):.1f}" '
        f'width="{sx(2.4) - sx(1.4):.1f}" height="{sy_val(-0.8) - sy_val(-5.0):.1f}" '
        f'fill="{_hex("green2")}" rx="2"/>',
        "",
        "    <!-- Spinach / cucumber (top center) -->",
        f'    <rect x="{sx(-1.8):.1f}" y="{sy_val(-5.0):.1f}" '
        f'width="{sx(1.8) - sx(-1.8):.1f}" height="{sy_val(-1.5) - sy_val(-5.0):.1f}" '
        f'fill="{_hex("green1")}" rx="2"/>',
        "",
        "  </g>",
        "",
        "  <!-- Sesame seeds -->",
        '  <g opacity="0.75">',
    ]

    for seed_x, seed_y in seeds:
        svg_parts.append(
            f'    <circle cx="{seed_x:.1f}" cy="{seed_y:.1f}" r="3.5" fill="{_hex("sesame")}"/>'
        )

    svg_parts.extend([
        "  </g>",
        "</svg>",
    ])

    svg_text = "\n".join(svg_parts)
    with open(path, "w") as f:
        f.write(svg_text)
    print(f"  SVG → {path}")


# ── Main ───────────────────────────────────────────────────────────

if __name__ == "__main__":
    root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    web_root = os.path.join(os.path.dirname(root), "kimbap-web")

    svg_path = os.path.join(root, "docs", "logo.svg")
    ico_path = os.path.join(web_root, "src", "app", "favicon.ico")

    # Allow overrides via CLI args
    if len(sys.argv) > 1:
        svg_path = sys.argv[1]
    if len(sys.argv) > 2:
        ico_path = sys.argv[2]

    os.makedirs(os.path.dirname(svg_path), exist_ok=True)
    os.makedirs(os.path.dirname(ico_path), exist_ok=True)

    print("Generating kimbap assets…")
    generate_svg(svg_path)
    generate_ico(ico_path)
    print("Done.")
