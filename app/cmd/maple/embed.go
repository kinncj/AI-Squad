package main

import "embed"

// embeddedTemplate holds the MAPLE template tree copied into projects by `maple init`.
// app/cmd/maple/template is a symlink to ../../../template for development; go:embed
// cannot follow symlinks, so the build (make build-app) replaces it with a real copy
// before building and restores the symlink after. See the Makefile build-app target.
//
//go:embed all:template
var embeddedTemplate embed.FS
