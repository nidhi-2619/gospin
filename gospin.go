package gospin

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Version is the gospin release version.
const Version = "1.0.0"

// EventKind classifies a progress Event emitted during Generate.
type EventKind int

const (
	// EventInfo marks the start of a generation phase (Message is the title).
	EventInfo EventKind = iota
	// EventDir reports that a directory was created (Path is set).
	EventDir
	// EventFile reports that a file was written (Path is set).
	EventFile
	// EventCmd reports a shell command being run (Message is the command line).
	EventCmd
	// EventWarn reports a non-fatal warning (Message is the detail).
	EventWarn
)

// Event is a single progress notification emitted by Generate when
// Options.OnEvent is set. The library writes nothing to stdout/stderr itself;
// rendering is entirely the caller's responsibility.
type Event struct {
	Kind    EventKind
	Path    string // relative path, for EventDir / EventFile
	Message string // detail, for EventInfo / EventCmd / EventWarn
}

// Options configures a call to Generate.
type Options struct {
	// Module is the Go module path of the generated project, e.g.
	// "github.com/acme/backend". Required.
	Module string

	// OutputDir is the directory the project is written to. If empty, the
	// last path segment of Module is used (see ResolvedOutputDir).
	OutputDir string

	// SkipInit, when true, skips running `go mod init`, `go mod tidy`, and
	// `go fmt` in the generated project. Useful in CI or offline.
	SkipInit bool

	// OnEvent, if non-nil, receives progress events as generation proceeds.
	// It is called synchronously from the goroutine that calls Generate.
	OnEvent func(Event)
}

// ResolvedOutputDir returns the directory Generate will write to, applying the
// default (last segment of Module) when OutputDir is empty.
func (o Options) ResolvedOutputDir() string {
	if o.OutputDir != "" {
		return o.OutputDir
	}
	m := strings.TrimRight(strings.TrimSpace(o.Module), "/")
	segs := strings.Split(m, "/")
	return segs[len(segs)-1]
}

func (o Options) emit(e Event) {
	if o.OnEvent != nil {
		o.OnEvent(e)
	}
}

// Generate scaffolds a complete, opinionated Go backend project on disk
// according to opts. It is safe to call into an existing directory; files are
// created or overwritten under the resolved output directory.
//
// Generate returns an error only for fatal problems (missing module path,
// filesystem failures). A failing `go mod tidy` — e.g. when offline — is
// reported as an EventWarn and does not fail generation.
func Generate(opts Options) error {
	module := strings.TrimSpace(opts.Module)
	if module == "" {
		return fmt.Errorf("gospin: Module is required")
	}
	outDir := opts.ResolvedOutputDir()

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("gospin: create output dir %q: %w", outDir, err)
	}

	opts.emit(Event{Kind: EventInfo, Message: "Creating directory structure"})
	dirs := []string{
		"cmd/api", "cmd/migrate", "bootstrap", "api/health", "server",
		"config", "common", "infra/db", "infra/cache", "infra/jwt",
		"middleware", "model", "repository", "utils", "constants",
		"mocks", "docs",
	}
	for _, d := range dirs {
		full := filepath.Join(outDir, d)
		if err := os.MkdirAll(full, 0o755); err != nil {
			return fmt.Errorf("gospin: create dir %q: %w", full, err)
		}
		opts.emit(Event{Kind: EventDir, Path: d + "/"})
	}

	opts.emit(Event{Kind: EventInfo, Message: "Writing source files"})
	for path, content := range Files(module) {
		if err := writeFile(outDir, path, content); err != nil {
			return err
		}
		opts.emit(Event{Kind: EventFile, Path: path})
	}

	// Keep otherwise-empty layer directories under version control.
	gitkeeps := []string{
		"model/.gitkeep", "repository/.gitkeep", "utils/.gitkeep",
		"constants/.gitkeep", "mocks/.gitkeep", "docs/.gitkeep",
	}
	for _, gk := range gitkeeps {
		if err := writeFile(outDir, gk, ""); err != nil {
			return err
		}
	}

	if opts.SkipInit {
		opts.emit(Event{Kind: EventWarn, Message: "skipping go mod init / tidy / fmt (SkipInit)"})
		return nil
	}

	opts.emit(Event{Kind: EventInfo, Message: "Initialising Go module"})
	for _, args := range [][]string{
		{"go", "mod", "init", module},
		{"go", "mod", "tidy"},
		{"go", "fmt", "./..."},
	} {
		runInDir(outDir, opts, args...)
	}
	return nil
}

// writeFile writes content to outDir/relPath, creating parent directories.
func writeFile(outDir, relPath, content string) error {
	full := filepath.Join(outDir, relPath)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fmt.Errorf("gospin: mkdir for %q: %w", full, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		return fmt.Errorf("gospin: write %q: %w", full, err)
	}
	return nil
}

// runInDir runs a command in dir. Failures are reported as warnings rather than
// returned, mirroring gospin's tolerance for offline `go mod tidy`.
func runInDir(dir string, opts Options, args ...string) {
	label := strings.Join(args, " ")
	opts.emit(Event{Kind: EventCmd, Message: label})
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		opts.emit(Event{Kind: EventWarn, Message: label + " failed: " + err.Error()})
	}
}
