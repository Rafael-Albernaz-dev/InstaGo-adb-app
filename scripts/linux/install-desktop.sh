#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
APP_ID="dev.rafaelalbernaz.instago"
BIN_PATH="${1:-$ROOT_DIR/build/bin/instago-gtk3}"

if [ ! -x "$BIN_PATH" ]; then
  echo "Binary not found or not executable: $BIN_PATH" >&2
  echo "Run 'make build-gtk3' (or 'make build') first." >&2
  exit 1
fi

ICON_SVG="$ROOT_DIR/assets/appicon.svg"
ICON_PNG="$ROOT_DIR/assets/appicon.png"
XDG_DATA_HOME="${XDG_DATA_HOME:-$HOME/.local/share}"
APPS_DIR="$XDG_DATA_HOME/applications"
ICONS_BASE="$XDG_DATA_HOME/icons/hicolor"

mkdir -p "$APPS_DIR" "$ICONS_BASE/scalable/apps"
cp "$ICON_SVG" "$ICONS_BASE/scalable/apps/$APP_ID.svg"

python3 - "$ICON_SVG" "$ICON_PNG" "$ICONS_BASE" "$APP_ID" <<'PY'
import os
import sys

svg_path, png_path, icons_base, app_id = sys.argv[1:5]
sizes = [48, 64, 128, 256, 512]

rendered = False
try:
    import gi
    gi.require_version("Rsvg", "2.0")
    from gi.repository import Rsvg
    import cairo

    handle = Rsvg.Handle.new_from_file(svg_path)
    rect = Rsvg.Rectangle()
    rect.x = 0.0
    rect.y = 0.0
    for size in sizes:
        out_dir = os.path.join(icons_base, f"{size}x{size}", "apps")
        os.makedirs(out_dir, exist_ok=True)
        surface = cairo.ImageSurface(cairo.FORMAT_ARGB32, size, size)
        ctx = cairo.Context(surface)
        rect.width = float(size)
        rect.height = float(size)
        handle.render_document(ctx, rect)
        surface.write_to_png(os.path.join(out_dir, f"{app_id}.png"))
    rendered = True
except Exception:
    rendered = False

if not rendered:
    from PIL import Image
    with Image.open(png_path) as im:
        for size in sizes:
            out_dir = os.path.join(icons_base, f"{size}x{size}", "apps")
            os.makedirs(out_dir, exist_ok=True)
            im.resize((size, size), Image.Resampling.LANCZOS).save(
                os.path.join(out_dir, f"{app_id}.png")
            )
PY

DESKTOP_FILE="$APPS_DIR/$APP_ID.desktop"
cat >"$DESKTOP_FILE" <<EOF
[Desktop Entry]
Version=1.0
Type=Application
Name=InstaGo
GenericName=Estúdio de Marketing
Comment=Estúdio criativo, biblioteca e agente IA para Instagram
Exec=$BIN_PATH
Path=$ROOT_DIR
Icon=$APP_ID
Terminal=false
Categories=Graphics;Utility;
StartupNotify=true
StartupWMClass=$APP_ID
EOF
chmod 644 "$DESKTOP_FILE"

if command -v gtk-update-icon-cache >/dev/null 2>&1; then
  gtk-update-icon-cache -f -t "$ICONS_BASE" >/dev/null 2>&1 || true
fi
if command -v update-desktop-database >/dev/null 2>&1; then
  update-desktop-database "$APPS_DIR" >/dev/null 2>&1 || true
fi

echo "Installed desktop entry: $DESKTOP_FILE"
echo "Installed icons under:   $ICONS_BASE"
