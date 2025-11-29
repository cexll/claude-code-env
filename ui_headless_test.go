package main

import (
	"os"
	"testing"
)

// Forces headless path by replacing stdout with a pipe and stdin with a pipe.
func TestSelectEnvironmentHeadless(t *testing.T) {
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdout pipe: %v", err)
	}
	rIn, wIn, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create stdin pipe: %v", err)
	}

	origStdout := os.Stdout
	origStdin := os.Stdin
	os.Stdout = wOut
	os.Stdin = rIn
	defer func() {
		os.Stdout = origStdout
		os.Stdin = origStdin
		rOut.Close()
		wOut.Close()
		rIn.Close()
		wIn.Close()
	}()

	config := Config{
		Environments: []Environment{
			{Name: "env1", URL: "https://api1.example.com", APIKey: "sk-ant-api03-a"},
			{Name: "env2", URL: "https://api2.example.com", APIKey: "sk-ant-api03-b"},
		},
	}

	env, err := selectEnvironmentWithArrows(config)
	if err != nil {
		t.Fatalf("expected headless selection to succeed: %v", err)
	}
	if env.Name != "env1" {
		t.Fatalf("expected first environment in headless mode, got %s", env.Name)
	}
}
