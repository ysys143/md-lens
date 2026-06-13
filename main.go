// Command mdl is a rich terminal markdown viewer: a `cat` for Markdown, with ANSI styling,
// syntax-highlighted code, inline images, and large-font headings.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	xterm "golang.org/x/term"

	"github.com/benelog/md-lens/internal/cli"
	"github.com/benelog/md-lens/internal/render"
	"github.com/benelog/md-lens/internal/term"
)

const version = "1.0.1"

func main() {
	opts, err := cli.Parse(os.Args[1:])
	if err != nil {
		fmt.Fprintln(os.Stderr, "mdl: "+err.Error())
		fmt.Fprintln(os.Stderr, "Try 'mdl --help'.")
		os.Exit(2)
	}

	switch {
	case opts.Help:
		printHelp()
		return
	case opts.Version:
		fmt.Println("mdl " + version)
		return
	case opts.Caps:
		caps := term.Detect(opts.Plain, opts.NoColor, opts.NoImages, opts.Width, opts.ForceGraphics)
		fmt.Print(term.FormatReport(caps, !opts.NoHeadingImages))
		return
	}

	// Bare `mdl` on an interactive terminal: nothing to read from stdin, so show usage
	// (with the current version) instead of hanging while waiting for input.
	if opts.File == "" && xterm.IsTerminal(int(os.Stdin.Fd())) {
		printHelp()
		return
	}

	markdown, baseDir, err := readInput(opts.File)
	if err != nil {
		name := "<stdin>"
		if opts.File != "" {
			name = opts.File
		}
		fmt.Fprintln(os.Stderr, "mdl: cannot read "+name+": "+err.Error())
		os.Exit(1)
	}

	caps := term.Detect(opts.Plain, opts.NoColor, opts.NoImages, opts.Width, opts.ForceGraphics)
	render.NewRenderer(caps, opts, baseDir).Render(markdown, os.Stdout)
}

// readInput reads the markdown source and resolves the base directory for relative image paths.
func readInput(file string) (markdown, baseDir string, err error) {
	if file == "" {
		data, rerr := io.ReadAll(os.Stdin)
		if rerr != nil {
			return "", "", rerr
		}
		cwd, _ := os.Getwd()
		return string(data), cwd, nil
	}
	data, rerr := os.ReadFile(file)
	if rerr != nil {
		return "", "", rerr
	}
	baseDir = "."
	if abs, aerr := filepath.Abs(file); aerr == nil {
		baseDir = filepath.Dir(abs)
	} else if cwd, cerr := os.Getwd(); cerr == nil {
		baseDir = cwd
	}
	return string(data), baseDir, nil
}

func printHelp() {
	fmt.Print("mdl v" + version + ` — a rich terminal markdown viewer

Usage: mdl [options] [file.md]
       (reads stdin when no file is given)

Options:
  -w, --width N             force output width in columns
      --no-color            disable ANSI color
      --no-images           do not render images
      --no-heading-images   render headings as styled text, not font images
      --force-kitty         force the kitty graphics protocol
      --force-iterm         force the iTerm2 inline image protocol
      --force-halfblock     force the unicode half-block image fallback
  -p, --plain               plain text, no styling
      --caps                show detected terminal capabilities and exit
  -h, --help                show this help
  -V, --version             show version
`)
}
