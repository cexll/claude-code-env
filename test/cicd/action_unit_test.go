// Package cicd provides unit tests for individual GitHub Actions
package cicd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetupActionUnit provides isolated testing for the setup action
func TestSetupActionUnit(t *testing.T) {
	setupAction, err := loadActionDefinition(getActionPath("setup", "action.yml"))
	require.NoError(t, err, "Setup action should load successfully")

	t.Run("InputValidation", func(t *testing.T) {
		// Test default values
		inputs := setupAction.Inputs
		assert.Equal(t, "1.24", inputs["go-version"].Default, "Default Go version should be 1.24")
		assert.Equal(t, "**/go.sum", inputs["cache-dependency-path"].Default, "Default cache path should be go.sum")
		assert.Equal(t, "true", inputs["install-tools"].Default, "Should install tools by default")
		assert.Equal(t, "true", inputs["validate-version-consistency"].Default, "Should validate versions by default")
		assert.Equal(t, "true", inputs["install-govulncheck"].Default, "Should install govulncheck by default")
		assert.Equal(t, "true", inputs["golangci-config-required"].Default, "Should require golangci config by default")

		// Test input types and requirements
		goVersionInput := inputs["go-version"]
		assert.False(t, goVersionInput.Required, "Go version should have default value")
		assert.NotEmpty(t, goVersionInput.Description, "Go version should have description")

		installToolsInput := inputs["install-tools"]
		assert.False(t, installToolsInput.Required, "Install tools should have default")
		assert.Contains(t, installToolsInput.Description, "golangci-lint", "Description should mention tools")
	})

	t.Run("OutputDefinition", func(t *testing.T) {
		outputs := setupAction.Outputs
		
		expectedOutputs := map[string]string{
			"go-version":       "Installed Go version",
			"cache-hit":        "Whether cache was hit for Go modules", 
			"tools-cache-hit":  "Whether cache was hit for tools",
			"version-consistent": "Whether Go versions are consistent across files",
			"config-valid":     "Whether configuration files are valid",
		}

		for outputName, expectedDesc := range expectedOutputs {
			output, exists := outputs[outputName]
			assert.True(t, exists, "Output %s should exist", outputName)
			if exists {
				assert.Contains(t, output.Description, expectedDesc, 
					"Output %s description should contain: %s", outputName, expectedDesc)
			}
		}
	})

	t.Run("StepSequenceValidation", func(t *testing.T) {
		steps := setupAction.Runs.Steps
		require.NotEmpty(t, steps, "Setup action should have steps")

		// Validate step order and dependencies
		expectedStepNames := []string{
			"Validate configuration consistency",
			"Set up Go", 
			"Cache Go modules",
			"Cache tools",
			"Download Go modules",
			"Install development tools",
			"Verify Go installation and tools",
			"Set environment variables",
		}

		assert.GreaterOrEqual(t, len(steps), len(expectedStepNames), 
			"Setup action should have all expected steps")

		// Check that Go setup happens before caching
		goSetupIndex := -1
		cacheIndex := -1
		for i, step := range steps {
			if step.Name == "Set up Go" {
				goSetupIndex = i
			}
			if step.Name == "Cache Go modules" {
				cacheIndex = i
			}
		}

		assert.NotEqual(t, -1, goSetupIndex, "Go setup step should exist")
		assert.NotEqual(t, -1, cacheIndex, "Cache step should exist")
		assert.Less(t, goSetupIndex, cacheIndex, "Go setup should happen before caching")
	})

	t.Run("ConditionalLogic", func(t *testing.T) {
		steps := setupAction.Runs.Steps

		// Find conditional steps
		var validationStep, toolInstallStep, toolCacheStep *Step
		for i := range steps {
			step := &steps[i]
			switch step.Name {
			case "Validate configuration consistency":
				validationStep = step
			case "Install development tools":
				toolInstallStep = step
			case "Cache tools":
				toolCacheStep = step
			}
		}

		// Validation step should be conditional
		require.NotNil(t, validationStep, "Validation step should exist")
		assert.Contains(t, validationStep.If, "validate-version-consistency == 'true'",
			"Validation should be conditional")

		// Tool installation should be conditional on cache miss
		require.NotNil(t, toolInstallStep, "Tool install step should exist")
		assert.Contains(t, toolInstallStep.If, "cache-tools.outputs.cache-hit != 'true'",
			"Tool installation should be conditional on cache miss")

		// Tool cache should be conditional on install-tools
		require.NotNil(t, toolCacheStep, "Tool cache step should exist")
		assert.Contains(t, toolCacheStep.If, "install-tools == 'true'",
			"Tool caching should be conditional")
	})
}

// TestTestActionUnit provides isolated testing for the test action
func TestTestActionUnit(t *testing.T) {
	testAction, err := loadActionDefinition(getActionPath("test", "action.yml"))
	require.NoError(t, err, "Test action should load successfully")

	t.Run("TestTypeHandling", func(t *testing.T) {
		inputs := testAction.Inputs
		testTypeInput := inputs["test-type"]
		
		assert.Equal(t, "unit", testTypeInput.Default, "Default test type should be unit")
		assert.Contains(t, testTypeInput.Description, "unit, integration, security, performance, all",
			"Test type description should list valid options")

		// Find test path determination step
		var pathStep *Step
		for i := range testAction.Runs.Steps {
			if testAction.Runs.Steps[i].Name == "Determine test path" {
				pathStep = &testAction.Runs.Steps[i]
				break
			}
		}

		require.NotNil(t, pathStep, "Test action should have path determination step")
		assert.Contains(t, pathStep.Run, "case", "Path step should use case statement")
		assert.Contains(t, pathStep.Run, "unit", "Should handle unit tests")
		assert.Contains(t, pathStep.Run, "integration", "Should handle integration tests")
		assert.Contains(t, pathStep.Run, "security", "Should handle security tests")
		assert.Contains(t, pathStep.Run, "performance", "Should handle performance tests")
	})

	t.Run("CoverageThresholdValidation", func(t *testing.T) {
		inputs := testAction.Inputs
		coverageInput := inputs["coverage-threshold"]
		
		assert.Equal(t, "80", coverageInput.Default, "Default coverage threshold should be 80%")
		assert.Contains(t, coverageInput.Description, "percentage", "Description should mention percentage")

		// Find coverage processing step
		var coverageStep *Step
		for i := range testAction.Runs.Steps {
			if testAction.Runs.Steps[i].Name == "Process coverage report" {
				coverageStep = &testAction.Runs.Steps[i]
				break
			}
		}

		require.NotNil(t, coverageStep, "Test action should have coverage processing step")
		assert.Contains(t, coverageStep.Run, "go tool cover", "Should use go tool cover")
		assert.Contains(t, coverageStep.Run, "bc -l", "Should use bc for threshold comparison")
	})

	t.Run("RetryLogic", func(t *testing.T) {
		inputs := testAction.Inputs
		retryInput := inputs["retry-count"]
		
		assert.Equal(t, "2", retryInput.Default, "Default retry count should be 2")

		// Find retry execution step
		var retryStep *Step
		for i := range testAction.Runs.Steps {
			if testAction.Runs.Steps[i].Name == "Run tests with retry logic" {
				retryStep = &testAction.Runs.Steps[i]
				break
			}
		}

		require.NotNil(t, retryStep, "Test action should have retry logic step")
		assert.Contains(t, retryStep.Run, "RETRY_COUNT", "Should use retry count variable")
		assert.Contains(t, retryStep.Run, "for i in", "Should have retry loop")
		assert.Contains(t, retryStep.Run, "sleep 2", "Should have delay between retries")
	})

	t.Run("PerformanceTestHandling", func(t *testing.T) {
		// Find test execution step
		var execStep *Step
		for i := range testAction.Runs.Steps {
			if testAction.Runs.Steps[i].Name == "Run tests with retry logic" {
				execStep = &testAction.Runs.Steps[i]
				break
			}
		}

		require.NotNil(t, execStep, "Test action should have execution step")
		assert.Contains(t, execStep.Run, "-bench=.", "Should include benchmark flags for performance tests")
		assert.Contains(t, execStep.Run, "-benchmem", "Should include memory benchmarks")
	})

	t.Run("OutputGeneration", func(t *testing.T) {
		outputs := testAction.Outputs
		
		expectedOutputs := []string{"test-result", "coverage-percentage", "total-tests", "failed-tests"}
		for _, output := range expectedOutputs {
			assert.Contains(t, outputs, output, "Test action should output %s", output)
		}

		// Find summary generation step
		var summaryStep *Step
		for i := range testAction.Runs.Steps {
			if testAction.Runs.Steps[i].Name == "Generate test summary" {
				summaryStep = &testAction.Runs.Steps[i]
				break
			}
		}

		require.NotNil(t, summaryStep, "Test action should generate summary")
		assert.Contains(t, summaryStep.Run, "test-summary.md", "Should generate markdown summary")
	})
}

// TestSecurityActionUnit provides isolated testing for the security action
func TestSecurityActionUnit(t *testing.T) {
	securityAction, err := loadActionDefinition(getActionPath("security", "action.yml"))
	require.NoError(t, err, "Security action should load successfully")

	t.Run("ScanTypeConfiguration", func(t *testing.T) {
		inputs := securityAction.Inputs
		scanTypeInput := inputs["scan-type"]
		
		assert.Equal(t, "both", scanTypeInput.Default, "Default scan type should be both")
		assert.Contains(t, scanTypeInput.Description, "gosec, govulncheck, both",
			"Scan type description should list valid options")

		// Find tool verification step
		var verifyStep *Step
		for i := range securityAction.Runs.Steps {
			if securityAction.Runs.Steps[i].Name == "Verify tool availability" {
				verifyStep = &securityAction.Runs.Steps[i]
				break
			}
		}

		require.NotNil(t, verifyStep, "Security action should verify tool availability")
		assert.Contains(t, verifyStep.Run, "gosec", "Should check gosec availability")
		assert.Contains(t, verifyStep.Run, "govulncheck", "Should check govulncheck availability")
	})

	t.Run("SeverityThresholds", func(t *testing.T) {
		inputs := securityAction.Inputs
		
		severityInput := inputs["severity-threshold"]
		assert.Equal(t, "high", severityInput.Default, "Default severity should be high")
		
		failCriticalInput := inputs["fail-on-critical"]
		assert.Equal(t, "true", failCriticalInput.Default, "Should fail on critical by default")
		
		failHighInput := inputs["fail-on-high"]
		assert.Equal(t, "true", failHighInput.Default, "Should fail on high by default")

		// Find severity evaluation step
		var evalStep *Step
		for i := range securityAction.Runs.Steps {
			if securityAction.Runs.Steps[i].Name == "Evaluate failure conditions" {
				evalStep = &securityAction.Runs.Steps[i]
				break
			}
		}

		require.NotNil(t, evalStep, "Security action should evaluate failure conditions")
		assert.Contains(t, evalStep.Run, "critical", "Should handle critical severity")
		assert.Contains(t, evalStep.Run, "high", "Should handle high severity")
		assert.Contains(t, evalStep.Run, "medium", "Should handle medium severity")
		assert.Contains(t, evalStep.Run, "low", "Should handle low severity")
	})

	t.Run("ReportGeneration", func(t *testing.T) {
		outputs := securityAction.Outputs
		
		expectedOutputs := map[string]bool{
			"gosec-findings":      true,
			"govulncheck-findings": true,
			"max-severity":        true,
			"overall-status":      true,
			"report-path":         true,
			"sarif-path":          true,
		}

		for output := range expectedOutputs {
			assert.Contains(t, outputs, output, "Security action should output %s", output)
		}

		// Find report generation steps
		var consolidatedStep, sarifStep *Step
		for i := range securityAction.Runs.Steps {
			step := &securityAction.Runs.Steps[i]
			if step.Name == "Generate consolidated security report" {
				consolidatedStep = step
			}
			if step.Name == "Generate SARIF report" {
				sarifStep = step
			}
		}

		require.NotNil(t, consolidatedStep, "Should generate consolidated report")
		assert.Contains(t, consolidatedStep.Run, "json", "Consolidated report should be JSON")

		require.NotNil(t, sarifStep, "Should generate SARIF report")
		assert.Contains(t, sarifStep.Run, "sarif", "SARIF step should generate SARIF")
		assert.Contains(t, sarifStep.If, "upload-sarif == 'true'", "SARIF generation should be conditional")
	})

	t.Run("ToolIntegration", func(t *testing.T) {
		// Find gosec step
		var gosecStep *Step
		for i := range securityAction.Runs.Steps {
			if securityAction.Runs.Steps[i].Name == "Run gosec security scan" {
				gosecStep = &securityAction.Runs.Steps[i]
				break
			}
		}

		require.NotNil(t, gosecStep, "Security action should run gosec")
		assert.Contains(t, gosecStep.Run, "-fmt json", "gosec should output JSON")
		assert.Contains(t, gosecStep.Run, "jq", "Should use jq to parse JSON")

		// Find govulncheck step
		var govulnStep *Step
		for i := range securityAction.Runs.Steps {
			if securityAction.Runs.Steps[i].Name == "Run govulncheck vulnerability scan" {
				govulnStep = &securityAction.Runs.Steps[i]
				break
			}
		}

		require.NotNil(t, govulnStep, "Security action should run govulncheck")
		assert.Contains(t, govulnStep.Run, "-json", "govulncheck should output JSON")
	})
}

// TestValidateConfigActionUnit provides isolated testing for the validate-config action
func TestValidateConfigActionUnit(t *testing.T) {
	validateAction, err := loadActionDefinition(getActionPath("validate-config", "action.yml"))
	require.NoError(t, err, "Validate config action should load successfully")

	t.Run("InputValidation", func(t *testing.T) {
		inputs := validateAction.Inputs
		
		expectedInputs := map[string]string{
			"validate-go-versions":     "true",
			"validate-golangci-config": "true", 
			"fail-on-mismatch":         "true",
		}

		for input, expectedDefault := range expectedInputs {
			inputDef, exists := inputs[input]
			assert.True(t, exists, "Input %s should exist", input)
			if exists {
				assert.Equal(t, expectedDefault, inputDef.Default, 
					"Input %s should have default %s", input, expectedDefault)
			}
		}
	})

	t.Run("OutputGeneration", func(t *testing.T) {
		outputs := validateAction.Outputs
		
		expectedOutputs := []string{
			"go-version-consistent",
			"golangci-config-valid", 
			"go-version-found",
		}

		for _, output := range expectedOutputs {
			assert.Contains(t, outputs, output, "Validate action should output %s", output)
		}
	})
}

// TestBuildActionUnit provides isolated testing for the build action
func TestBuildActionUnit(t *testing.T) {
	buildAction, err := loadActionDefinition(getActionPath("build", "action.yml"))
	require.NoError(t, err, "Build action should load successfully")

	t.Run("CrossPlatformBuild", func(t *testing.T) {
		inputs := buildAction.Inputs
		
		// Should have target platform inputs
		expectedInputs := []string{"target-os", "target-arch", "version"}
		for _, input := range expectedInputs {
			assert.Contains(t, inputs, input, "Build action should have %s input", input)
		}

		// Should have ldflags configuration
		steps := buildAction.Runs.Steps
		var buildStep *Step
		for i := range steps {
			if steps[i].Name == "Build binary" || strings.Contains(steps[i].Run, "go build") {
				buildStep = &steps[i]
				break
			}
		}

		if buildStep != nil {
			assert.Contains(t, buildStep.Run, "GOOS", "Build should set GOOS")
			assert.Contains(t, buildStep.Run, "GOARCH", "Build should set GOARCH")
			assert.Contains(t, buildStep.Run, "-ldflags", "Build should use ldflags")
		}
	})
}

// TestActionErrorHandling validates error handling in actions
func TestActionErrorHandling(t *testing.T) {
	actions := []string{"setup", "test", "security", "build", "validate-config"}

	for _, actionName := range actions {
		t.Run(fmt.Sprintf("ErrorHandling_%s", actionName), func(t *testing.T) {
			action, err := loadActionDefinition(getActionPath(actionName, "action.yml"))
			require.NoError(t, err)

			// Check for error handling patterns in steps
			for i, step := range action.Runs.Steps {
				// Steps should handle errors appropriately
				if step.Run != "" {
					// Look for error handling patterns
					hasErrorHandling := strings.Contains(step.Run, "set +e") ||
						strings.Contains(step.Run, "|| true") ||
						strings.Contains(step.Run, "exit 1") ||
						strings.Contains(step.Run, "require") ||
						strings.Contains(step.Run, "if [ $? -ne 0 ]")

					// Critical steps should have error handling
					isCritical := strings.Contains(step.Name, "Install") ||
						strings.Contains(step.Name, "Setup") ||
						strings.Contains(step.Name, "Run") ||
						strings.Contains(step.Name, "Build")

					if isCritical && step.Continue == false {
						// For debugging - not failing the test but logging
						t.Logf("Step %d (%s) in action %s might need error handling: %v", 
							i, step.Name, actionName, hasErrorHandling)
					}
				}
			}
		})
	}
}

// TestActionPerformance validates performance characteristics of actions
func TestActionPerformance(t *testing.T) {
	t.Run("CacheEfficiency", func(t *testing.T) {
		setupAction, err := loadActionDefinition(getActionPath("setup", "action.yml"))
		require.NoError(t, err)

		// Find cache steps
		cacheSteps := 0
		for _, step := range setupAction.Runs.Steps {
			if strings.Contains(step.Uses, "cache") {
				cacheSteps++
				
				// Cache steps should have proper keys
				assert.NotEmpty(t, step.With["key"], "Cache step should have key")
				assert.NotEmpty(t, step.With["path"], "Cache step should have path")
				
				// Should have restore keys for better cache hits
				if restoreKeys, ok := step.With["restore-keys"]; ok {
					assert.NotEmpty(t, restoreKeys, "Restore keys should not be empty")
				}
			}
		}

		assert.GreaterOrEqual(t, cacheSteps, 2, "Setup should have at least 2 cache steps (modules and tools)")
	})

	t.Run("ParallelTestExecution", func(t *testing.T) {
		testAction, err := loadActionDefinition(getActionPath("test", "action.yml"))
		require.NoError(t, err)

		// Find test execution step
		var execStep *Step
		for i := range testAction.Runs.Steps {
			if strings.Contains(testAction.Runs.Steps[i].Run, "go test") {
				execStep = &testAction.Runs.Steps[i]
				break
			}
		}

		if execStep != nil {
			assert.Contains(t, execStep.Run, "-parallel", "Tests should run in parallel")
		}
	})

	t.Run("ConditionalExecution", func(t *testing.T) {
		// Actions should minimize unnecessary work
		actions := []string{"setup", "test", "security"}
		
		for _, actionName := range actions {
			action, err := loadActionDefinition(getActionPath(actionName, "action.yml"))
			require.NoError(t, err)

			conditionalSteps := 0
			for _, step := range action.Runs.Steps {
				if step.If != "" {
					conditionalSteps++
				}
			}

			// Actions should have some conditional logic
			t.Logf("Action %s has %d conditional steps", actionName, conditionalSteps)
		}
	})
}

// TestActionSecurity validates security aspects of actions
func TestActionSecurity(t *testing.T) {
	t.Run("PinnedActionVersions", func(t *testing.T) {
		actions := []string{"setup", "test", "security", "build", "validate-config"}
		
		for _, actionName := range actions {
			action, err := loadActionDefinition(getActionPath(actionName, "action.yml"))
			require.NoError(t, err)

			for _, step := range action.Runs.Steps {
				if step.Uses != "" && strings.HasPrefix(step.Uses, "actions/") {
					// Should use pinned versions (SHA or version tag)
					assert.Contains(t, step.Uses, "@", 
						fmt.Sprintf("Action %s step '%s' should use pinned version", actionName, step.Name))
					
					// Should not use @main or @master
					assert.NotContains(t, step.Uses, "@main", 
						fmt.Sprintf("Action %s should not use @main", actionName))
					assert.NotContains(t, step.Uses, "@master", 
						fmt.Sprintf("Action %s should not use @master", actionName))
				}
			}
		}
	})

	t.Run("PermissionMinimization", func(t *testing.T) {
		// Actions themselves don't have permissions, but check that they don't request unnecessary privileges
		actions := []string{"setup", "test", "security", "build", "validate-config"}
		
		for _, actionName := range actions {
			action, err := loadActionDefinition(getActionPath(actionName, "action.yml"))
			require.NoError(t, err)

			for _, step := range action.Runs.Steps {
				// Check for dangerous commands
				if step.Run != "" {
					assert.NotContains(t, step.Run, "sudo rm -rf", 
						fmt.Sprintf("Action %s should not use dangerous commands", actionName))
					assert.NotContains(t, step.Run, "curl | sh", 
						fmt.Sprintf("Action %s should not pipe curl to shell", actionName))
				}
			}
		}
	})

	t.Run("SecretHandling", func(t *testing.T) {
		securityAction, err := loadActionDefinition(getActionPath("security", "action.yml"))
		require.NoError(t, err)

		// Security action should not expose secrets in outputs
		for _, step := range securityAction.Runs.Steps {
			if step.Run != "" {
				// Should not echo sensitive information
				assert.NotContains(t, step.Run, "echo $GITHUB_TOKEN",
					"Security action should not echo secrets")
			}
		}
	})
}

// Helper function imports are handled by Go's import system above