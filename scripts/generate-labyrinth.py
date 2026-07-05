#!/usr/bin/env python3
"""Generate 7-circuit circular maze SVG groups for labyrinth-aura.

Each wall lies on a concentric circle with a doorway gap. Rings rotate about
the shared centre so doorways shift while the figure always reads as a maze.

Run: python3 scripts/generate-labyrinth.py
"""

from __future__ import annotations

import math

CX, CY = 100, 100
N_CIRCUITS = 7
R_OUTER = 86
R_HUB = 10
GAP_DEG = 26
# Outermost entrance at bottom (90°), alternate right/left inward
BASE_GAPS = [90, 0, 180, 0, 180, 0, 180]
DURATIONS = [56, 44, 68, 52, 60, 48, 72]
DIRECTIONS = ["normal", "reverse", "normal", "reverse", "normal", "reverse", "normal"]
OFFSETS = [0, 18, -12, 24, -8, 15, -20]


def pt(r: float, deg: float) -> tuple[float, float]:
    rad = math.radians(deg)
    return (CX + r * math.cos(rad), CY + r * math.sin(rad))


def ring_radius(index: int) -> float:
    return R_OUTER - index * (R_OUTER - R_HUB - 4) / (N_CIRCUITS - 1 + 0.5)


def arc_path(r: float, gap_deg: float) -> str:
    a1 = gap_deg + GAP_DEG / 2
    a2 = gap_deg - GAP_DEG / 2
    x0, y0 = pt(r, a1)
    x1, y1 = pt(r, a2)
    sweep = ((a2 - a1) % 360) or 360
    large = 1 if sweep > 180 else 0
    return f"M {x0:.2f} {y0:.2f} A {r:.2f} {r:.2f} 0 {large} 1 {x1:.2f} {y1:.2f}"


def ring_style(index: int) -> str:
    return (
        f"--ring-duration: {DURATIONS[index]}s; "
        f"--ring-offset: {OFFSETS[index]}deg; "
        f"--ring-direction: {DIRECTIONS[index]};"
    )


def main() -> None:
    radii = [ring_radius(i) for i in range(N_CIRCUITS)]
    print("<!-- 7-circuit circular maze: rotating concentric wall rings -->")

    for i in range(N_CIRCUITS):
        d = arc_path(radii[i], BASE_GAPS[i])
        print(f'      <g class="labyrinth-ring labyrinth-ring-{i}" style="{ring_style(i)}">')
        print(f'        <path class="labyrinth-wall" d="{d}"/>')
        print("      </g>")

    radials = [
        (180, 1, 3),
        (180, 3, 5),
        (0, 2, 4),
        (0, 4, 6),
    ]
    for idx, (bearing, ri, rj) in enumerate(radials):
        r_inner = radii[max(ri, rj)]
        r_outer = radii[min(ri, rj)]
        x1, y1 = pt(r_inner, bearing)
        x2, y2 = pt(r_outer, bearing)
        parent = min(ri, rj)
        print(f'      <g class="labyrinth-ring labyrinth-radial-{idx}" style="{ring_style(parent)}">')
        print(f'        <path class="labyrinth-wall" d="M {x1:.2f} {y1:.2f} L {x2:.2f} {y2:.2f}"/>')
        print("      </g>")

    print(
        '      <circle class="labyrinth-hub" cx="100" cy="100" r="8" '
        'fill="url(#labyrinth-hub-glow)" stroke="url(#labyrinth-stroke)" stroke-width="2"/>'
    )


if __name__ == "__main__":
    main()
