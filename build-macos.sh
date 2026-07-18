#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")"
BIN="dist/lclub-macos"
mkdir -p dist

go mod tidy

# --- Build a universal binary (Intel + Apple Silicon) ---
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/.lclub-amd64 .
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o dist/.lclub-arm64 .
lipo -create -output "$BIN" dist/.lclub-amd64 dist/.lclub-arm64
rm -f dist/.lclub-amd64 dist/.lclub-arm64
chmod +x "$BIN"

# Re-sign ad hoc: lipo invalidates the per-slice signature and unsigned
# arm64 binaries are refused by macOS on Apple Silicon.
codesign --force -s - "$BIN" 2>/dev/null || true

# --- Create a native .icns from the shared PNG icon ---
ICONSET="dist/.icon.iconset"
rm -rf "$ICONSET"
mkdir -p "$ICONSET"
for s in 16 32 128 256 512; do
    d=$((s * 2))
    sips -z "$s" "$s" assets/ui/icon.png --out "$ICONSET/icon_${s}x${s}.png"     >/dev/null
    sips -z "$d" "$d" assets/ui/icon.png --out "$ICONSET/icon_${s}x${s}@2x.png"  >/dev/null
done
iconutil -c icns "$ICONSET" -o dist/.lclub.icns
rm -rf "$ICONSET"

# --- Attach the icon to the executable so Finder shows it ---
# Rez/DeRez/SetFile ship with the Xcode Command Line Tools.
if command -v Rez >/dev/null 2>&1 && command -v DeRez >/dev/null 2>&1 && command -v SetFile >/dev/null 2>&1; then
    DeRez -only icns dist/.lclub.icns > dist/.lclub.rsrc
    Rez -append dist/.lclub.rsrc -o "$BIN"
    SetFile -a C "$BIN"
    rm -f dist/.lclub.rsrc dist/.lclub.icns
else
    mv dist/.lclub.icns dist/lclub.icns
    echo "note: Rez/SetFile not found (install Xcode Command Line Tools);"
    echo "      Finder icon was not attached, dist/lclub.icns is kept instead."
    echo "      The Dock icon still works: it is embedded in the binary from icon.png."
fi

printf 'Done: %s\n' "$BIN"