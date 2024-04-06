package main

import (
	"io"
	"log"
	"os"

	"github.com/fatih/color"
	"github.com/urfave/cli/v2"
)

type coloredWriter struct {
	Color  *color.Color
	Writer io.Writer
}

func (cw coloredWriter) Write(data []byte) (int, error) {
	return cw.Writer.Write([]byte(cw.Color.Sprintf(string(data))))
}

func getLogger(ctx *cli.Context, colorAttrs ...color.Attribute) *log.Logger {
	if len(colorAttrs) == 0 {
		colorAttrs = []color.Attribute{color.FgYellow}
	}

	wr := coloredWriter{
		Color:  color.New(colorAttrs...),
		Writer: os.Stderr,
	}

	if ctx != nil {
		wr.Writer = ctx.App.ErrWriter
	}

	return log.New(&wr, "", log.Ldate|log.Ltime|log.Lshortfile)
}
