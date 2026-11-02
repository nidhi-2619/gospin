// Package gospin generates production-ready, opinionated Go backend projects.
//
// It writes a complete "Layered Root Layout" project to disk — Gin HTTP server,
// PostgreSQL connection pool, Uber Zap logging, typed env config, a working
// /health endpoint with tests, Dockerfile, docker-compose, and a Makefile — so
// you can skip the boilerplate and start writing real code immediately.
//
// # As a library
//
//	err := gospin.Generate(gospin.Options{
//		Module:    "github.com/acme/backend",
//		OutputDir: "./backend",
//	})
//
// Generate writes nothing to stdout/stderr; pass Options.OnEvent to observe
// progress. Use Files to obtain the generated content as an in-memory map
// without touching disk.
//
// # As a command
//
// The companion command lives at github.com/nidhi-2619/gospin/cmd/gospin:
//
//	go install github.com/nidhi-2619/gospin/cmd/gospin@latest
//	gospin github.com/acme/backend
package gospin
