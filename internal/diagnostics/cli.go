package diagnostics

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/srnoob2570/better-rpt/internal/config"
	"github.com/srnoob2570/better-rpt/internal/i18n"
)

// Main is the command-line entry point. The first return value reports whether
// the arguments were a diagnostics command; the second is the process exit code.
func Main(args []string, stdout, stderr io.Writer, version string) (bool, int) {
	if !hasDiagnose(args) {
		return false, 0
	}
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}
	return true, runCLI(args, stdout, stderr, version, func(ctx context.Context, options Options) (Report, error) {
		return Run(ctx, options), nil
	})
}

func hasDiagnose(args []string) bool {
	for _, arg := range args {
		if arg == "--diagnose" || strings.HasPrefix(arg, "--diagnose=") {
			return true
		}
	}
	return false
}

type runner func(context.Context, Options) (Report, error)

func runCLI(args []string, stdout, stderr io.Writer, version string, run runner) int {
	flags := flag.NewFlagSet("better-rpt --diagnose", flag.ContinueOnError)
	flags.SetOutput(stderr)
	diagnose := flags.Bool("diagnose", false, "run read-only diagnostics and exit")
	asJSON := flags.Bool("json", false, "print the report as JSON")
	out := flags.String("out", "", "also save the report to this path")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if !*diagnose {
		fmt.Fprintln(stderr, "error: --diagnose is required")
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintf(stderr, "error: unexpected arguments: %v\n", flags.Args())
		return 2
	}

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()
	paths, pathsErr := config.ResolvePaths()
	report, err := run(ctx, Options{
		Version:  version,
		Language: string(i18n.Detect()),
		Deps:     DefaultDeps(paths, pathsErr),
	})
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	var data []byte
	if *asJSON {
		data, err = RenderJSON(report)
		if err == nil {
			err = ValidateJSON(data)
		}
		if err != nil {
			fmt.Fprintf(stderr, "error: render JSON: %v\n", err)
			return 1
		}
	} else {
		data = []byte(RenderMarkdown(report))
	}
	if _, err := stdout.Write(data); err != nil {
		fmt.Fprintf(stderr, "error: write report: %v\n", err)
		return 1
	}
	if *out != "" {
		if err := os.WriteFile(*out, data, 0o600); err != nil {
			fmt.Fprintf(stderr, "error: save report: %v\n", err)
			return 2
		}
	}
	if report.Summary.Counts.Fail > 0 {
		return 1
	}
	return 0
}
