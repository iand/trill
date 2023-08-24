package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/iand/pontium/hlog"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slog"
)

var app = &cli.App{
	Name:  "trill",
	Usage: "a toolkit for working with the IPFS DHT",
	Commands: []*cli.Command{
		findNodeCommand,
	},
}

func main() {
	log.SetFlags(log.LstdFlags | log.LUTC)
	ctx := context.Background()

	if err := app.RunContext(ctx, os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}

func setupLogging(cc *cli.Context) {
	logLevel := new(slog.LevelVar)
	logLevel.Set(slog.LevelWarn)

	if loggingOpts.Verbose {
		logLevel.Set(slog.LevelInfo)
	}

	if loggingOpts.VeryVerbose {
		logLevel.Set(slog.LevelDebug)
	}

	h := new(hlog.Handler).WithLevel(logLevel.Level())
	slog.SetDefault(slog.New(h))
}

var loggingFlags = []cli.Flag{
	&cli.BoolFlag{
		Name:        "verbose",
		Aliases:     []string{"v"},
		Usage:       "Set logging level more verbose to include info level logs",
		Value:       true,
		Destination: &loggingOpts.Verbose,
	},

	&cli.BoolFlag{
		Name:        "veryverbose",
		Aliases:     []string{"vv"},
		Usage:       "Set logging level more verbose to include debug level logs",
		Destination: &loggingOpts.VeryVerbose,
	},
}

func mergeFlags(flagsets ...[]cli.Flag) []cli.Flag {
	flags := flagsets[0]
	for i := 1; i < len(flagsets); i++ {
		flags = append(flags, flagsets[i]...)
	}
	return flags
}

var loggingOpts struct {
	Verbose     bool
	VeryVerbose bool
	Hlog        bool
}
