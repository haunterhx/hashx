package main

import (
	"fmt"
	"io"
	"os"
	"strings"
)

// bannerArt is the HASHX logo rendered in the FIGlet "standard" font.
// Kept as plain ASCII so it renders correctly in any terminal or log sink.
const bannerArt = ` _   _    _    ____  _   ___  __
| | | |  / \  / ___|| | | \ \/ /
| |_| | / _ \ \___ \| |_| |\  /
|  _  |/ ___ \ ___) |  _  |/  \
|_| |_/_/   \_\____/|_| |_/_/\_\
`

// Build metadata. Override at build time, e.g.:
//
//	go build -ldflags "-X main.Version=1.2.0 -X main.Commit=$(git rev-parse --short HEAD)"
var (
	Version = "dev"
	Commit  = "none"
	Tagline = "hash identification & analysis"
)

const (
	ansiCyan  = "\033[36m"
	ansiDim   = "\033[2m"
	ansiReset = "\033[0m"
)

// Banner returns the banner text. When color is true, ANSI escape codes are added.
func Banner(color bool) string {
	art := strings.TrimRight(bannerArt, "\n")
	width := 0
	for _, line := range strings.Split(art, "\n") {
		if len(line) > width {
			width = len(line)
		}
	}

	info := fmt.Sprintf("%s | v%s (%s)", Tagline, Version, Commit)
	rule := strings.Repeat("-", max(width, len(info)))

	if !color {
		return fmt.Sprintf("%s\n%s\n%s\n", art, rule, info)
	}
	return fmt.Sprintf("%s%s%s\n%s%s\n%s%s\n",
		ansiCyan, art, ansiReset,
		ansiDim, rule, info, ansiReset)
}

// PrintBanner writes the banner to w. Color is enabled only when w is a
// terminal and NO_COLOR is unset (https://no-color.org).
func PrintBanner(w io.Writer) {
	fmt.Fprint(w, Banner(useColor(w)))
}

func useColor(w io.Writer) bool {
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// stdoutColor is evaluated once; the table is written to stdout.
var stdoutColor = useColor(os.Stdout)

func colorize(code, s string) string {
	if !stdoutColor {
		return s
	}
	return code + s + ansiReset
}
