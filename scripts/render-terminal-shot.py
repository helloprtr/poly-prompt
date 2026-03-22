#!/usr/bin/env python3

from __future__ import annotations

import argparse
import math
from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter, ImageFont


FONT_CANDIDATES = [
    "/System/Library/Fonts/AppleSDGothicNeo.ttc",
    "/System/Library/Fonts/Supplemental/Menlo.ttc",
    "/System/Library/Fonts/Supplemental/Andale Mono.ttf",
    "/usr/share/fonts/truetype/noto/NotoSansMonoCJK-Regular.ttc",
    "/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
    "/usr/share/fonts/TTF/DejaVuSansMono.ttf",
]


def load_font(size: int) -> ImageFont.FreeTypeFont | ImageFont.ImageFont:
    for candidate in FONT_CANDIDATES:
        path = Path(candidate)
        if not path.exists():
            continue
        try:
            return ImageFont.truetype(str(path), size=size)
        except OSError:
            continue
    return ImageFont.load_default()


def text_width(font: ImageFont.ImageFont, text: str) -> float:
    if hasattr(font, "getlength"):
        return float(font.getlength(text))
    box = font.getbbox(text)
    return float(box[2] - box[0])


def wrap_lines(font: ImageFont.ImageFont, lines: list[str], max_width: int) -> list[str]:
    wrapped: list[str] = []
    for raw_line in lines:
        line = raw_line.expandtabs(2)
        if line == "":
            wrapped.append("")
            continue
        current = ""
        for ch in line:
            candidate = current + ch
            if current and text_width(font, candidate) > max_width:
                wrapped.append(current)
                current = ch
            else:
                current = candidate
        wrapped.append(current)
    return wrapped


def render_terminal(title: str, body: str, output: Path, width: int) -> None:
    body_font = load_font(22)
    title_font = load_font(20)

    outer_margin = 36
    card_radius = 28
    title_bar_height = 54
    body_padding_x = 34
    body_padding_y = 28
    footer_height = 22

    max_text_width = width - (outer_margin * 2) - (body_padding_x * 2)
    wrapped = wrap_lines(body_font, body.splitlines(), max_text_width)
    ascent, descent = body_font.getmetrics()
    line_height = ascent + descent + 8
    body_height = max(1, len(wrapped)) * line_height
    card_height = title_bar_height + body_padding_y + body_height + body_padding_y + footer_height
    canvas_height = card_height + (outer_margin * 2)

    canvas = Image.new("RGBA", (width, canvas_height), "#0b1020")
    shadow = Image.new("RGBA", (width, canvas_height), (0, 0, 0, 0))
    shadow_draw = ImageDraw.Draw(shadow)
    shadow_draw.rounded_rectangle(
        (
            outer_margin,
            outer_margin + 8,
            width - outer_margin,
            canvas_height - outer_margin + 8,
        ),
        radius=card_radius,
        fill=(2, 6, 23, 180),
    )
    shadow = shadow.filter(ImageFilter.GaussianBlur(18))
    canvas.alpha_composite(shadow)

    draw = ImageDraw.Draw(canvas)
    card_box = (
        outer_margin,
        outer_margin,
        width - outer_margin,
        canvas_height - outer_margin,
    )
    draw.rounded_rectangle(card_box, radius=card_radius, fill="#0f172a", outline="#334155", width=1)
    draw.rounded_rectangle(
        (card_box[0], card_box[1], card_box[2], card_box[1] + title_bar_height),
        radius=card_radius,
        fill="#111827",
    )
    draw.rectangle(
        (card_box[0], card_box[1] + title_bar_height - 24, card_box[2], card_box[1] + title_bar_height),
        fill="#111827",
    )

    dot_y = outer_margin + 18
    for index, color in enumerate(("#fb7185", "#fbbf24", "#34d399")):
        x = outer_margin + 18 + index * 18
        draw.ellipse((x, dot_y, x + 12, dot_y + 12), fill=color)

    title_x = outer_margin + 90
    draw.text((title_x, outer_margin + 14), title, font=title_font, fill="#cbd5e1")

    text_x = outer_margin + body_padding_x
    text_y = outer_margin + title_bar_height + body_padding_y
    for index, line in enumerate(wrapped):
        draw.text((text_x, text_y + index * line_height), line, font=body_font, fill="#e5e7eb")

    footer = f"{len(wrapped)} lines"
    footer_width = text_width(title_font, footer)
    draw.text(
        (width - outer_margin - body_padding_x - footer_width, canvas_height - outer_margin - footer_height),
        footer,
        font=title_font,
        fill="#64748b",
    )

    output.parent.mkdir(parents=True, exist_ok=True)
    canvas.convert("RGB").save(output, format="PNG")


def main() -> None:
    parser = argparse.ArgumentParser(description="Render a terminal-like PNG screenshot from a text file.")
    parser.add_argument("--title", required=True)
    parser.add_argument("--input", required=True)
    parser.add_argument("--output", required=True)
    parser.add_argument("--width", type=int, default=1480)
    args = parser.parse_args()

    body = Path(args.input).read_text()
    render_terminal(args.title, body, Path(args.output), args.width)


if __name__ == "__main__":
    main()
