#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")"
mkdir -p dist

go mod tidy
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o dist/lclub-linux .
cp assets/ui/icon.png dist/lclub.png
cat > dist/LClub.desktop <<'EOF'
[Desktop Entry]
Type=Application
Name=LClub
Comment=Find all pairs
Exec=LClub
Icon=lclub
Terminal=false
Categories=Game;
EOF
printf 'Done: dist/LClub\n'
