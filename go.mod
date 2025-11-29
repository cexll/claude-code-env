module github.com/cexll/claude-code-env

go 1.23.0

toolchain go1.24.5

require golang.org/x/term v0.33.0

replace golang.org/x/term => ./third_party/golang.org/x/term
