// Command gospin scaffolds a production-ready Go backend project.
//
// Usage:
//
//	gospin -module github.com/org/project [-output ./myapp]
//	gospin github.com/org/project
//	gospin --version
//
// The generator itself lives in the importable package
// github.com/nidhi-2619/gospin; this command is a thin CLI over it.
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/nidhi-2619/gospin"
)

// ─── Terminal Styling ─────────────────────────────────────────────────────────

const (
	reset  = "\033[0m"
	bold   = "\033[1m"
	dim    = "\033[2m"
	red    = "\033[31m"
	green  = "\033[32m"
	yellow = "\033[33m"
	blue   = "\033[34m"
	cyan   = "\033[36m"
)

func main() {
	var (
		flagModule   string
		flagOutput   string
		flagSkipInit bool
		flagVersion  bool
	)

	flag.StringVar(&flagModule, "module", "", "Go module path (e.g. github.com/org/project)")
	flag.StringVar(&flagOutput, "output", "", "Output directory (default: last segment of module path)")
	flag.BoolVar(&flagSkipInit, "skip-init", false, "Skip go mod init / tidy / fmt")
	flag.BoolVar(&flagVersion, "version", false, "Print version and exit")
	flag.Usage = func() {
		fmt.Fprintln(os.Stderr, bold+"gospin"+reset+" — Go backend project scaffolder v"+gospin.Version)
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Usage:")
		fmt.Fprintln(os.Stderr, "  gospin -module <module> [-output <dir>] [-skip-init]")
		fmt.Fprintln(os.Stderr, "  gospin <module>")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Flags:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Examples:")
		fmt.Fprintln(os.Stderr, "  gospin github.com/acme/backend")
		fmt.Fprintln(os.Stderr, "  gospin -module github.com/acme/api -output ./services/api")
	}
	flag.Parse()

	if flagVersion {
		fmt.Printf("gospin v%s\n", gospin.Version)
		return
	}

	if flagModule == "" && flag.NArg() > 0 {
		flagModule = flag.Arg(0)
	}
	if flagModule == "" {
		fmt.Fprintln(os.Stderr, red+bold+"error:"+reset+" -module is required")
		fmt.Fprintln(os.Stderr, "")
		flag.Usage()
		os.Exit(1)
	}

	opts := gospin.Options{
		Module:    flagModule,
		OutputDir: flagOutput,
		SkipInit:  flagSkipInit,
		OnEvent:   render,
	}
	outDir := opts.ResolvedOutputDir()

	printBanner()
	logKV("Module", flagModule)
	logKV("Output", outDir)

	if err := gospin.Generate(opts); err != nil {
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, red+bold+"error:"+reset+" "+err.Error())
		os.Exit(1)
	}

	printSuccess(flagModule, outDir)
}

// render maps library progress events to styled terminal output.
func render(e gospin.Event) {
	switch e.Kind {
	case gospin.EventInfo:
		fmt.Printf("\n%s%s▶ %s%s\n", bold, blue, e.Message, reset)
	case gospin.EventDir, gospin.EventFile:
		fmt.Printf("  %s%s✓%s %s\n", bold, green, reset, e.Path)
	case gospin.EventCmd:
		fmt.Printf("  %s$ %s%s\n", dim, e.Message, reset)
	case gospin.EventWarn:
		fmt.Printf("  %s%s⚠  %s%s\n", bold, yellow, e.Message, reset)
	}
}

// ─── CLI Presentation ──────────────────────────────────────────────────────────

func printBanner() {
	fmt.Println()
	fmt.Println(bold + cyan + "  ┌─────────────────────────────────┐" + reset)
	fmt.Println(bold + cyan + "  │   gospin v" + gospin.Version + "              │" + reset)
	fmt.Println(bold + cyan + "  │   Go Backend Project Spinner  │" + reset)
	fmt.Println(bold + cyan + "  └─────────────────────────────────┘" + reset)
	fmt.Println()
}

func printSuccess(module, outDir string) {
	_ = module
	fmt.Println()
	fmt.Println(bold + green + "  ✓ Project scaffolded successfully!" + reset)
	fmt.Println()
	fmt.Println(dim + "  Next steps:" + reset)
	fmt.Printf("    %scd %s%s\n", bold, outDir, reset)
	fmt.Printf("    %scp .env.example .env%s       # edit your credentials\n", bold, reset)
	fmt.Printf("    %smake docker-up%s             # start postgres\n", bold, reset)
	fmt.Printf("    %smake run%s                   # start the API\n", bold, reset)
	fmt.Printf("    %scurl localhost:8080/health%s\n", bold, reset)
	fmt.Println()
}

func logKV(k, v string) {
	fmt.Printf("  %s%-8s%s %s%s%s\n", dim, k+":", reset, bold, v, reset)
}
