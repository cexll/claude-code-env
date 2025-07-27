package main

import (
	"os"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"
)

// TestCrossPlatformCompatibility tests platform-specific functionality
func TestCrossPlatformCompatibility(t *testing.T) {
	t.Run("terminal detection across platforms", func(t *testing.T) {
		caps := detectTerminalCapabilities()
		
		// Basic validation that should work on all platforms
		if caps.Width <= 0 || caps.Height <= 0 {
			t.Error("Terminal dimensions should be positive")
		}
		
		// Platform-specific tests
		switch runtime.GOOS {
		case "windows":
			t.Log("Windows-specific terminal detection")
			// Windows might have different behavior
		case "darwin":
			t.Log("macOS-specific terminal detection")
			// macOS Terminal.app behavior
		case "linux":
			t.Log("Linux-specific terminal detection")
			// Various Linux terminal emulators
		default:
			t.Logf("Unknown platform: %s", runtime.GOOS)
		}
	})

	t.Run("file permissions across platforms", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "cce-platform-test-*")
		if err != nil {
			t.Fatalf("Failed to create temp directory: %v", err)
		}
		defer os.RemoveAll(tempDir)

		testFile := tempDir + "/test-permissions.json"
		err = os.WriteFile(testFile, []byte("{}"), 0600)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}

		// Check file permissions
		info, err := os.Stat(testFile)
		if err != nil {
			t.Fatalf("Failed to stat test file: %v", err)
		}

		// On Unix-like systems, permissions should be exactly 0600
		if runtime.GOOS != "windows" {
			if info.Mode().Perm() != 0600 {
				t.Errorf("Expected 0600 permissions, got %v", info.Mode().Perm())
			}
		}
	})

	t.Run("signal handling preparation", func(t *testing.T) {
		// Test that signal-related functionality doesn't panic
		// This is preparation for interrupt handling during terminal operations
		
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Signal handling preparation panicked: %v", r)
			}
		}()
		
		// Basic syscall operations should work
		fd := int(syscall.Stdin)
		if fd < 0 {
			t.Error("Invalid stdin file descriptor")
		}
	})
}

// TestTerminalCompatibilityMatrix tests various terminal environment combinations
func TestTerminalCompatibilityMatrix(t *testing.T) {
	// Test different TERM environment variable combinations
	termVariants := []struct {
		term        string
		description string
		expectANSI  bool
	}{
		{"xterm-256color", "Modern xterm with 256 colors", true},
		{"xterm-color", "Xterm with color support", true},
		{"xterm", "Basic xterm", true},
		{"screen-256color", "Screen with 256 colors", true},
		{"screen", "Basic screen", true},
		{"tmux-256color", "Tmux with 256 colors", true},
		{"tmux", "Basic tmux", true},
		{"linux", "Linux console", true},
		{"vt100", "VT100 terminal", true},
		{"vt52", "VT52 terminal (very old)", false},
		{"dumb", "Dumb terminal", false},
		{"", "No TERM set", false},
	}

	originalTerm := os.Getenv("TERM")
	defer os.Setenv("TERM", originalTerm)

	for _, tv := range termVariants {
		t.Run("TERM_"+tv.term, func(t *testing.T) {
			if tv.term == "" {
				os.Unsetenv("TERM")
			} else {
				os.Setenv("TERM", tv.term)
			}

			caps := detectTerminalCapabilities()
			
			if caps.SupportsANSI != tv.expectANSI {
				t.Errorf("TERM=%s: expected ANSI support %v, got %v", 
					tv.term, tv.expectANSI, caps.SupportsANSI)
			}
			
			// Cursor support should generally match ANSI support
			if caps.SupportsCursor != caps.SupportsANSI {
				t.Errorf("TERM=%s: cursor support (%v) should match ANSI support (%v)",
					tv.term, caps.SupportsCursor, caps.SupportsANSI)
			}
		})
	}
}

// TestSSHEnvironmentDetection tests behavior in SSH sessions
func TestSSHEnvironmentDetection(t *testing.T) {
	t.Run("SSH session simulation", func(t *testing.T) {
		// Save original SSH environment
		originalSSH := os.Getenv("SSH_CONNECTION")
		originalSSHTTY := os.Getenv("SSH_TTY")
		
		defer func() {
			if originalSSH == "" {
				os.Unsetenv("SSH_CONNECTION")
			} else {
				os.Setenv("SSH_CONNECTION", originalSSH)
			}
			if originalSSHTTY == "" {
				os.Unsetenv("SSH_TTY")
			} else {
				os.Setenv("SSH_TTY", originalSSHTTY)
			}
		}()

		// Simulate SSH environment
		os.Setenv("SSH_CONNECTION", "192.168.1.1 12345 192.168.1.100 22")
		os.Setenv("SSH_TTY", "/dev/pts/0")

		caps := detectTerminalCapabilities()
		
		// SSH environments should still support basic terminal features
		// but may have limitations
		t.Logf("SSH simulation - Terminal capabilities: %+v", caps)
		
		// Basic dimensions should still be available
		if caps.Width <= 0 || caps.Height <= 0 {
			t.Error("SSH environment should still provide terminal dimensions")
		}
	})
}

// TestScreenTmuxEnvironments tests screen and tmux compatibility
func TestScreenTmuxEnvironments(t *testing.T) {
	environments := []struct {
		name     string
		termVar  string
		sty      string
		tmux     string
	}{
		{"screen", "screen", "12345.pts-0.hostname", ""},
		{"screen-256color", "screen-256color", "12345.pts-0.hostname", ""},
		{"tmux", "tmux-256color", "", "/tmp/tmux-1000/default,12345,0"},
		{"tmux-basic", "tmux", "", "/tmp/tmux-1000/default,12345,0"},
	}

	originalTerm := os.Getenv("TERM")
	originalSTY := os.Getenv("STY")
	originalTMUX := os.Getenv("TMUX")
	
	defer func() {
		os.Setenv("TERM", originalTerm)
		if originalSTY == "" {
			os.Unsetenv("STY")
		} else {
			os.Setenv("STY", originalSTY)
		}
		if originalTMUX == "" {
			os.Unsetenv("TMUX")
		} else {
			os.Setenv("TMUX", originalTMUX)
		}
	}()

	for _, env := range environments {
		t.Run(env.name, func(t *testing.T) {
			os.Setenv("TERM", env.termVar)
			
			if env.sty != "" {
				os.Setenv("STY", env.sty)
			} else {
				os.Unsetenv("STY")
			}
			
			if env.tmux != "" {
				os.Setenv("TMUX", env.tmux)
			} else {
				os.Unsetenv("TMUX")
			}

			caps := detectTerminalCapabilities()
			
			// Screen and tmux should generally support ANSI
			if !caps.SupportsANSI {
				t.Errorf("%s should support ANSI sequences", env.name)
			}
			
			// Should have reasonable dimensions
			if caps.Width < 20 || caps.Height < 5 {
				t.Errorf("%s dimensions too small: %dx%d", env.name, caps.Width, caps.Height)
			}
		})
	}
}

// TestCIEnvironmentDetection tests CI/CD environment detection
func TestCIEnvironmentDetection(t *testing.T) {
	ciEnvironments := []struct {
		name     string
		envVars  map[string]string
	}{
		{
			"GitHub Actions",
			map[string]string{
				"GITHUB_ACTIONS": "true",
				"CI":             "true",
			},
		},
		{
			"GitLab CI",
			map[string]string{
				"GITLAB_CI": "true",
				"CI":        "true",
			},
		},
		{
			"Jenkins",
			map[string]string{
				"JENKINS_URL": "http://jenkins.example.com",
				"BUILD_ID":    "123",
			},
		},
		{
			"Generic CI",
			map[string]string{
				"CI":                    "true",
				"CONTINUOUS_INTEGRATION": "true",
			},
		},
	}

	// Save original environment
	originalEnv := make(map[string]string)
	for _, ciEnv := range ciEnvironments {
		for key := range ciEnv.envVars {
			originalEnv[key] = os.Getenv(key)
		}
	}

	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	for _, ciEnv := range ciEnvironments {
		t.Run(ciEnv.name, func(t *testing.T) {
			// Clear all CI variables first
			for _, otherEnv := range ciEnvironments {
				for key := range otherEnv.envVars {
					os.Unsetenv(key)
				}
			}

			// Set specific CI environment
			for key, value := range ciEnv.envVars {
				os.Setenv(key, value)
			}

			if !isHeadlessMode() {
				t.Errorf("%s should be detected as headless mode", ciEnv.name)
			}
		})
	}
}

// TestPerformanceRequirements tests that enhancements meet performance requirements
func TestPerformanceRequirements(t *testing.T) {
	t.Run("startup overhead under 100ms", func(t *testing.T) {
		iterations := 10
		totalDuration := time.Duration(0)

		for i := 0; i < iterations; i++ {
			start := time.Now()
			
			// Test the expensive operations that would happen during startup
			_ = detectTerminalCapabilities()
			_ = newModelValidator()
			
			duration := time.Since(start)
			totalDuration += duration
		}

		averageDuration := totalDuration / time.Duration(iterations)
		maxAllowed := 100 * time.Millisecond

		if averageDuration > maxAllowed {
			t.Errorf("Average startup overhead %v exceeds limit of %v", 
				averageDuration, maxAllowed)
		}

		t.Logf("Average startup overhead: %v (limit: %v)", averageDuration, maxAllowed)
	})

	t.Run("memory footprint under 5% increase", func(t *testing.T) {
		// Baseline memory measurement
		runtime.GC()
		var m1 runtime.MemStats
		runtime.ReadMemStats(&m1)

		// Create enhanced components
		_ = detectTerminalCapabilities()
		validator := newModelValidator()
		errorCtx := newErrorContext("test", "component").
			addContext("key", "value").
			addSuggestion("suggestion")

		// Force retention
		_ = validator
		_ = errorCtx

		runtime.GC()
		var m2 runtime.MemStats
		runtime.ReadMemStats(&m2)

		// Calculate memory increase
		memIncrease := m2.HeapAlloc - m1.HeapAlloc
		maxIncrease := uint64(4096) // 4KB as reasonable limit for enhancements

		if memIncrease > maxIncrease {
			t.Errorf("Memory increase %d bytes exceeds limit of %d bytes", 
				memIncrease, maxIncrease)
		}

		t.Logf("Memory increase: %d bytes (limit: %d bytes)", memIncrease, maxIncrease)
	})
}

// TestErrorRecoveryIntegration tests integration of various error recovery mechanisms
func TestErrorRecoveryIntegration(t *testing.T) {
	t.Run("terminal state recovery integration", func(t *testing.T) {
		// Create a test config that would trigger terminal operations
		config := Config{
			Environments: []Environment{
				{Name: "test1", URL: "https://api.anthropic.com", APIKey: "sk-ant-test123456789"},
				{Name: "test2", URL: "https://api.anthropic.com", APIKey: "sk-ant-test123456789"},
			},
		}

		// This should not panic and should handle terminal state properly
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("Terminal integration panicked: %v", r)
			}
		}()

		// Test various fallback scenarios
		env, err := selectEnvironmentWithArrows(config)
		if err != nil {
			// In test environment, this might fail due to no stdin
			// but it should fail gracefully
			if !strings.Contains(err.Error(), "selection") {
				t.Errorf("Unexpected error type: %v", err)
			}
		} else {
			// If it succeeded, verify we got a valid environment
			if env.Name == "" {
				t.Error("Selected environment should have a name")
			}
		}
	})

	t.Run("end-to-end error propagation", func(t *testing.T) {
		// Test that errors propagate correctly through the system
		// with enhanced error context
		
		testCases := []struct {
			name     string
			args     []string
			contains []string
		}{
			{
				"invalid environment name",
				[]string{"--env", "nonexistent"},
				[]string{"not found"},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				err := handleCommand(tc.args)
				if err == nil {
					t.Error("Expected error for invalid command")
					return
				}

				errMsg := err.Error()
				for _, contains := range tc.contains {
					if !strings.Contains(errMsg, contains) {
						t.Errorf("Error message should contain '%s': %v", contains, err)
					}
				}
			})
		}
	})
}

// BenchmarkCrossPlatformOperations benchmarks platform-specific operations
func BenchmarkCrossPlatformOperations(b *testing.B) {
	b.Run("terminal_detection", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			detectTerminalCapabilities()
		}
	})

	b.Run("headless_detection", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			isHeadlessMode()
		}
	})

	b.Run("key_parsing", func(b *testing.B) {
		inputs := [][]byte{
			{0x1b, '[', 'A'},
			{'\n'},
			{'a'},
		}
		
		for i := 0; i < b.N; i++ {
			for _, input := range inputs {
				parseKeyInput(input)
			}
		}
	})
}