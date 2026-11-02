package gospin_test

import (
	"fmt"
	"sort"

	"github.com/nidhi-2619/gospin"
)

// Generate a project into a directory.
func ExampleGenerate() {
	err := gospin.Generate(gospin.Options{
		Module:    "github.com/acme/backend",
		OutputDir: "./backend",
		SkipInit:  true, // don't shell out to `go` in this example
	})
	if err != nil {
		fmt.Println("error:", err)
	}
}

// Inspect what would be generated without writing to disk.
func ExampleFiles() {
	files := gospin.Files("github.com/acme/backend")

	paths := make([]string, 0, len(files))
	for p := range files {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	fmt.Println(paths[0])
	fmt.Println(paths[1])
	// Output:
	// .env.example
	// .gitignore
}
