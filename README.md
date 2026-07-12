# LClub

**LClub** is a modern cross-platform remake of my classic DOS memory card game originally written in **Turbo Pascal** in the late 1990s.

The gameplay remains faithful to the original version while the implementation has been completely rewritten in **Go** using **Ebitengine**.

---

## Features

- Native desktop application
- Written entirely in Go
- No browser
- No web server
- Cross-platform (Windows / Linux / macOS)
- Hardware accelerated rendering
- Fullscreen support
- Smooth card animations
- Modern high-resolution artwork
- 18 unique animal cards
- Embedded assets (`go:embed`)
- Single executable after compilation

---

## Gameplay

The objective is simple:

1. The game creates a field of **36 cards (18 pairs)**.
2. All cards are shown for several seconds.
3. The cards are turned face down.
4. The player must remember their positions and find every matching pair.
5. The timer starts immediately after the cards are hidden.
6. The game ends when every pair has been found.

The faster the player finishes, the better.

---

## Controls

| Action         | Key / Mouse        |
|----------------|--------------------|
| Start game     | **Enter**          |
| New game       | **N**              |
| Fullscreen     | **F11**            |
| Exit           | **Esc**            |
| Open card      | Left Mouse Button  |
| Temporary hint | Right Mouse Button |

---

## Game Rules

- The board contains **18 unique animal pairs**.
- Every animal appears exactly twice.
- Matching cards remain opened.
- Incorrect pairs are automatically hidden after a short delay.
- During the initial preview all cards are visible.
- A countdown is displayed in the **top information bar** and never covers the playing field.

---

## Interface

Compared to the original DOS version, the interface has been redesigned:

- modern menu
- high-resolution cards
- scalable UI
- smooth animations
- improved layout
- fullscreen support
- large readable fonts

The original yellow **"LCLUB"** text has been replaced with a graphical logo.

---

## Project Structure

```
assets/
    cards/
        back.png
        card01.png
        ...
        card18.png

    ui/
        logo.png

font.go
main.go
README.md
go.mod
```

---

## Assets

All game resources are embedded directly into the executable using Go's embedded filesystem.

```go
//go:embed assets/**
```

No external files are required after compilation.

---

## Building

```go mod tidy
go mod build
```

## Dependencies

- Go 1.24+
- Ebitengine v2

---

## License

This project is a personal remake of my original school project.

Original DOS version:
Turbo Pascal (1998)

Modern version:
Go + Ebitengine