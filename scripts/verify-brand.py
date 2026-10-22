#!/usr/bin/env python3
"""Verify the Better RPT Relay Link brand kit and its app/package rollout.

The script only reads artifacts. It prints a single success marker after every
assertion for the requested mode passes. `selftest` exercises the negative
paths so an absence check cannot silently pass on a missing file.
"""

from __future__ import annotations

import argparse
import hashlib
import sys
import tempfile
import xml.etree.ElementTree as ET
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
BRAND = ROOT / "packaging" / "brand"
PACKAGING = ROOT / "packaging"


def rel(path: Path) -> str:
    try:
        return str(path.relative_to(ROOT))
    except ValueError:
        return str(path)

INK = "#0b1418"
AMBER = "#f4a261"
TEAL = "#2a9d8f"
CREAM = "#f8fafc"
BRAND_COLORS = {INK, AMBER, TEAL, CREAM}

SVG_NS = "{http://www.w3.org/2000/svg}"


class VerifyError(Exception):
    pass


def fail(message: str) -> None:
    raise VerifyError(message)


def require(path: Path) -> Path:
    if not path.is_file():
        fail(f"missing {rel(path)}")
    return path


def sha256(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def svg_root(path: Path) -> ET.Element:
    try:
        return ET.parse(path).getroot()
    except ET.ParseError as exc:
        fail(f"invalid SVG {rel(path)}: {exc}")


def svg_colors(root: ET.Element) -> set[str]:
    colors: set[str] = set()

    def add(value: str | None) -> None:
        if not value:
            return
        value = value.strip().lower()
        if value in {"none", "transparent", "currentcolor"} or value.startswith("url("):
            return
        colors.add(value)

    for node in root.iter():
        add(node.get("fill"))
        add(node.get("stroke"))
        add(node.get("stop-color"))
        style = node.get("style", "")
        for declaration in style.split(";"):
            if ":" not in declaration:
                continue
            key, value = declaration.split(":", 1)
            if key.strip() in {"fill", "stroke", "stop-color"}:
                add(value)

    return colors


def check_svg(path: Path, allowed: set[str] | None = None, single: str | None = None,
             min_paths: int = 0) -> ET.Element:
    root = svg_root(path)

    forbidden = {"text", "image", "foreignObject", "filter", "mask"}
    for node in root.iter():
        tag = node.tag.rsplit("}", 1)[-1]
        if tag in forbidden:
            fail(f"{rel(path)} contains <{tag}>")

    if not root.get("viewBox"):
        fail(f"{rel(path)} has no viewBox")

    paths = [node for node in root.iter() if node.tag == SVG_NS + "path"]
    if len(paths) < min_paths:
        fail(f"{rel(path)} has {len(paths)} paths, expected at least {min_paths}")

    colors = svg_colors(root)
    if single is not None:
        bad = {color for color in colors if color != single}
        if bad:
            fail(f"{rel(path)} is not one-colour: {sorted(bad)}")
    if allowed is not None:
        bad = {color for color in colors if color not in allowed}
        if bad:
            fail(f"{rel(path)} has unexpected colours: {sorted(bad)}")

    return root


def check_png(path: Path, size: int | None = None) -> Image.Image:
    require(path)
    try:
        image = Image.open(path)
        image.load()
    except Exception as exc:
        fail(f"cannot read PNG {rel(path)}: {exc}")
    if size is not None and image.size != (size, size):
        fail(f"{rel(path)} is {image.size}, expected {(size, size)}")
    rgba = image.convert("RGBA")
    alpha = rgba.getchannel("A")
    visible = rgba.width * rgba.height - alpha.histogram()[0]
    if visible < max(16, int(rgba.width * rgba.height * 0.02)):
        fail(f"{rel(path)} is effectively blank")
    return image


def check_ico(path: Path) -> None:
    require(path)
    try:
        image = Image.open(path)
    except Exception as exc:
        fail(f"cannot read icon {rel(path)}: {exc}")
    sizes = set(image.ico.sizes()) if hasattr(image, "ico") else set()
    for wanted in {(16, 16), (32, 32), (48, 48)}:
        if wanted not in sizes:
            fail(f"{rel(path)} lacks {wanted[0]}x{wanted[1]}")


def check_text(path: Path, needles: list[str]) -> None:
    require(path)
    text = path.read_text(encoding="utf-8").lower()
    for needle in needles:
        if needle.lower() not in text:
            fail(f"{rel(path)} lacks {needle!r}")


def check_kit() -> None:
    svgs = {
        "icon.svg": (BRAND_COLORS, None, 1),
        "icon-small.svg": (BRAND_COLORS, None, 1),
        "mark.svg": ({AMBER, TEAL, CREAM}, None, 3),
        "mark-light.svg": ({AMBER, TEAL, INK}, None, 3),
        "mark-mono.svg": ({INK}, INK, 3),
        "mark-white.svg": ({CREAM}, CREAM, 3),
        "wordmark.svg": ({INK}, INK, 1),
        "wordmark-white.svg": ({CREAM}, CREAM, 1),
        "lockup-horizontal.svg": (BRAND_COLORS, None, 4),
        "lockup-horizontal-dark.svg": (BRAND_COLORS, None, 4),
        "lockup-stacked.svg": (BRAND_COLORS, None, 4),
        "lockup-stacked-dark.svg": (BRAND_COLORS, None, 4),
        "lockup-horizontal-mono.svg": ({INK}, INK, 4),
        "lockup-stacked-mono.svg": ({INK}, INK, 4),
        "lockup-horizontal-white.svg": ({CREAM}, CREAM, 4),
        "lockup-stacked-white.svg": ({CREAM}, CREAM, 4),
    }
    for name, (allowed, single, min_paths) in svgs.items():
        check_svg(require(BRAND / name), allowed=set(allowed), single=single, min_paths=min_paths)

    for size in (16, 24, 32, 48, 64, 128, 256, 512):
        check_png(require(BRAND / f"icon-{size}.png"), size=size)
    check_png(require(BRAND / "icon-small-128.png"), size=128)
    for name in ("lockup-horizontal.png", "lockup-horizontal-dark.png"):
        image = check_png(require(BRAND / name))
        if image.width < 1200:
            fail(f"{name} is too narrow: {image.width}")
    for name in ("lockup-stacked.png", "lockup-stacked-dark.png"):
        image = check_png(require(BRAND / name))
        if image.width < 600:
            fail(f"{name} is too narrow: {image.width}")
    for size in (16, 32, 48):
        check_png(require(BRAND / f"favicon-{size}.png"), size=size)
    check_ico(require(BRAND / "favicon.ico"))

    board = check_png(require(BRAND / "brand-board.png"))
    if board.width < 1200 or board.height < 800:
        fail(f"brand-board.png is too small: {board.size}")

    check_text(require(BRAND / "PALETTE.md"), ["#0B1418", "#F4A261", "#2A9D8F", "#F8FAFC"])
    check_text(require(BRAND / "USAGE.md"), ["espacio de respeto", "tamaños mínimos", "no ", "marca registrada"])

    print("BRAND_KIT_OK")


def check_app_icons() -> None:
    icon_svg = require(BRAND / "icon.svg")
    packaged_svg = require(PACKAGING / "better-rpt.svg")
    if sha256(icon_svg) != sha256(packaged_svg):
        fail("packaging/better-rpt.svg is not the approved icon master")

    root = check_svg(packaged_svg, allowed=BRAND_COLORS, min_paths=1)
    if any(node.tag == SVG_NS + "circle" for node in root.iter()):
        fail("packaging/better-rpt.svg still contains the old ring geometry")

    check_png(require(PACKAGING / "better-rpt.png"), size=256)
    if sha256(PACKAGING / "better-rpt.png") != sha256(BRAND / "icon-256.png"):
        fail("packaging/better-rpt.png does not match brand/icon-256.png")

    for size in (16, 24, 32, 48, 64, 128, 256, 512):
        path = require(PACKAGING / "icons" / "hicolor" / f"{size}x{size}" / "apps" / "better-rpt.png")
        check_png(path, size=size)
        source = BRAND / f"icon-{size}.png"
        if source.is_file() and sha256(path) != sha256(source):
            fail(f"hicolor {size} icon does not match brand/icon-{size}.png")

    # The new mark is a horizontal relay link: teal left, amber right, cream play.
    image = Image.open(PACKAGING / "better-rpt.png").convert("RGBA")
    samples = {
        (44, 128): TEAL,
        (212, 128): AMBER,
        (128, 128): CREAM,
        (128, 16): INK,
    }
    for point, expected in samples.items():
        pixel = image.getpixel(point)
        got = "#%02x%02x%02x" % pixel[:3]
        if got != expected:
            fail(f"packaging/better-rpt.png pixel {point} is {got}, expected {expected}")

    print("APP_ICONS_OK")


def check_integration() -> None:
    brand = require(ROOT / "internal" / "brand" / "brand.go").read_text(encoding="utf-8")
    for needle in ("//go:embed better-rpt.png", "var Icon"):
        if needle not in brand:
            fail(f"internal/brand/brand.go lacks {needle!r}")
    check_png(require(ROOT / "frontend" / "public" / "brand" / "icon.png"), size=512)
    if sha256(ROOT / "frontend" / "public" / "brand" / "icon.png") != sha256(
        BRAND / "icon-512.png"
    ):
        fail("frontend/public/brand/icon.png does not match brand/icon-512.png")

    nfpm = require(PACKAGING / "nfpm.yaml").read_text(encoding="utf-8")
    for size in (16, 24, 32, 48, 64, 128, 256, 512):
        needle = f"/usr/share/icons/hicolor/{size}x{size}/apps/better-rpt.png"
        if needle not in nfpm:
            fail(f"packaging/nfpm.yaml lacks {needle}")

    desktop = require(PACKAGING / "better-rpt.desktop").read_text(encoding="utf-8")
    if "Icon=better-rpt" not in desktop:
        fail("packaging/better-rpt.desktop does not point at better-rpt")

    print("APP_INTEGRATION_OK")


def check_selftest() -> None:
    bad_cases = [
        ('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><text>Bad</text></svg>', None, None),
        ('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><image href="x.png"/></svg>', None, None),
        ('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 10 10"><path fill="#000000" d="M0 0h1v1z"/><path fill="#ffffff" d="M2 2h1v1z"/></svg>', {INK, CREAM}, INK),
    ]
    with tempfile.TemporaryDirectory() as tmp:
        for index, (raw, allowed, single) in enumerate(bad_cases):
            path = Path(tmp) / f"bad-{index}.svg"
            path.write_text(raw, encoding="utf-8")
            try:
                check_svg(path, allowed=allowed, single=single, min_paths=1)
            except VerifyError:
                continue
            fail(f"selftest case {index} was not rejected")

    print("BRAND_SELFTEST_OK")


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("mode", choices=("kit", "app-icons", "integration", "selftest"))
    args = parser.parse_args()
    try:
        {
            "kit": check_kit,
            "app-icons": check_app_icons,
            "integration": check_integration,
            "selftest": check_selftest,
        }[args.mode]()
    except VerifyError as exc:
        print(f"FAIL: {exc}", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
