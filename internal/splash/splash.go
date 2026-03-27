package splash

import (
	"fmt"
	"math"
	"strings"
)

func rgb(r, g, b int) string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", r, g, b)
}

const reset = "\x1b[0m"

var (
	nori1  = rgb(26, 74, 26)
	nori2  = rgb(46, 110, 46)
	rice1  = rgb(250, 248, 240)
	rice2  = rgb(240, 234, 216)
	sesame = rgb(200, 160, 48)
	green1 = rgb(52, 160, 52)
	green2 = rgb(42, 122, 42)
	orange = rgb(240, 120, 32)
	yolk1  = rgb(244, 208, 48)
	yolk2  = rgb(248, 240, 64)
	white  = rgb(253, 244, 240)
	red1   = rgb(232, 72, 72)
	red2   = rgb(224, 64, 64)

	colDim    = rgb(72, 79, 88)
	colMuted  = rgb(110, 118, 129)
	colWhite  = rgb(255, 255, 255)
	colPurple = rgb(163, 113, 247)
	colGreen  = rgb(63, 185, 80)
)

const (
	cols = 13
	rows = 11
	cx   = 6.5
	cy   = 5.5
	rx   = 5.6
	ry   = 4.8
)

type cell struct {
	color string
	glyph string
}

func getCell(nx, ny float64) *cell {
	px := nx * rx
	py := ny * ry
	d := math.Sqrt(nx*nx + ny*ny)

	if d > 1.00 {
		return nil
	}
	if d > 0.92 {
		return &cell{nori1, "██"}
	}
	if d > 0.84 {
		return &cell{nori2, "▒▒"}
	}

	if d > 0.52 {
		speckle := math.Abs(math.Sin(px*7.3+py*3.1)) > 0.92
		if speckle && d < 0.68 {
			return &cell{sesame, "··"}
		}
		if d > 0.70 {
			return &cell{rice1, "██"}
		}
		return &cell{rice2, "▓▓"}
	}

	switch {
	case py < -1.5 && math.Abs(px) < 1.8:
		return &cell{green1, "▓▓"}
	case py < -0.8 && math.Abs(px) < 2.4 && math.Abs(px) > 1.4:
		return &cell{green2, "▒▒"}
	case px < -1.0 && px > -2.4 && py > -1.5 && py < 0.3:
		return &cell{orange, "▓▓"}
	case py > -1.5 && py < -0.3 && math.Abs(px) < 1.0:
		return &cell{yolk1, "▓▓"}
	case py > -0.4 && py < 0.8 && math.Abs(px) < 1.2:
		return &cell{yolk2, "██"}
	case px > 0.9 && px < 2.4 && py > -1.5 && py < 0.9:
		if int(math.Floor((py+1.5)*1.6))%2 == 0 {
			return &cell{white, "░░"}
		}
		return &cell{red1, "▓▓"}
	case py > 0.7 && py < 2.0 && math.Abs(px) < 1.7:
		return &cell{red2, "▓▓"}
	}

	return &cell{rice2, "░░"}
}

type Mode string

const (
	ModeEmbedded  Mode = "embedded"
	ModeConnected Mode = "connected"
)

type Options struct {
	Version     string
	Mode        Mode
	VaultStatus string
	Server      string
}

func Render(opts Options) string {
	version := strings.TrimSpace(opts.Version)
	if version == "" {
		version = "dev"
	}

	mode := opts.Mode
	if mode != ModeConnected {
		mode = ModeEmbedded
	}

	vaultStatus := strings.TrimSpace(opts.VaultStatus)
	if vaultStatus == "" {
		vaultStatus = "ready"
	}

	var sb strings.Builder
	sb.Grow(rows*cols*16 + 256)
	sb.WriteByte('\n')

	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			nx := (float64(c) - cx) / rx
			ny := (float64(r) - cy) / ry
			v := getCell(nx, ny)
			if v != nil {
				sb.WriteString(v.color)
				sb.WriteString(v.glyph)
				sb.WriteString(reset)
			} else {
				sb.WriteString("  ")
			}
		}
		sb.WriteByte('\n')
	}

	bar := colDim + "────────────────────" + reset
	modeStr := colMuted + string(mode) + reset
	if mode == ModeConnected && strings.TrimSpace(opts.Server) != "" {
		modeStr = colMuted + "connected · " + colDim + strings.TrimSpace(opts.Server) + reset
	}

	sb.WriteByte('\n')
	sb.WriteString(" " + bar + "\n")
	sb.WriteString(" " + colWhite + "k i m b a p" + reset + "\n")
	sb.WriteString(" " + colMuted + "action runtime" + reset + colDim + " · " + reset + colPurple + version + reset + "\n")
	sb.WriteString(" " + bar + "\n")
	sb.WriteString(" " + colGreen + "vault" + reset + colMuted + " " + vaultStatus + " · " + reset + modeStr + colMuted + " · " + colDim + "kimbap.sh" + reset + "\n")

	return sb.String()
}

func Print(opts Options) {
	fmt.Print(Render(opts))
}
