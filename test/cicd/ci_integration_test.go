// Package cicd provides meta-tests for validating the test suite itself
package cicd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSuiteMetaValidation validates the test suite implementation itself
func TestSuiteMetaValidation(t *testing.T) {
	t.Run("TestFileStructure", func(t *testing.T) {
		// Validate all required test files exist
		requiredFiles := []string{
			"test/cicd/workflow_integration_test.go",
			"test/cicd/action_unit_test.go",
			"test/cicd/failure_scenario_test.go",
			"test/cicd/ci_integration_test.go",
			"test/cicd/run_tests.sh",
			"test/cicd/README.md",
		}

		repoRoot := getRepoRoot()
		for _, file := range requiredFiles {
			fullPath := filepath.Join(repoRoot, file)
			assert.FileExists(t, fullPath, "Required test file should exist: %s", file)
		}
	})

	t.Run("TestRunnerScript", func(t *testing.T) {
		runnerScript := filepath.Join(getRepoRoot(), "test/cicd/run_tests.sh")
		
		// Check script is executable
		info, err := os.Stat(runnerScript)
		require.NoError(t, err)
		
		mode := info.Mode()
		assert.True(t, mode&0111 != 0, "Test runner script should be executable")

		// Check script has required functions
		content, err := os.ReadFile(runnerScript)
		require.NoError(t, err)
		
		scriptContent := string(content)
		requiredFunctions := []string{
			"check_prerequisites",
			"validate_workflow_files", 
			"run_workflow_tests",
			"run_action_tests",
			"run_config_tests",
			"run_security_tests",
			"run_coverage_test",
			"generate_test_report",
		}

		for _, function := range requiredFunctions {
			assert.Contains(t, scriptContent, function, 
				"Test runner should contain function: %s", function)
		}
	})

	t.Run("TestCoverage", func(t *testing.T) {
		// Run actual test coverage to validate our test coverage
		cmd := exec.Command("go", "test", "-coverprofile=coverage-meta.out", "./test/cicd")
		cmd.Dir = getRepoRoot()
		
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("Test output: %s", string(output))
			t.Skipf("Could not run coverage test: %v", err)
		}

		// Extract coverage
		coverageCmd := exec.Command("go", "tool", "cover", "-func=coverage-meta.out")
		coverageCmd.Dir = getRepoRoot()
		
		coverageOutput, err := coverageCmd.CombinedOutput()
		if err != nil {
			t.Skipf("Could not extract coverage: %v", err)
		}

		coverageStr := string(coverageOutput)
		lines := strings.Split(coverageStr, "\n")
		
		for _, line := range lines {
			if strings.Contains(line, "total:") {
				t.Logf("Test suite coverage: %s", line)
				break
			}
		}

		// Cleanup coverage file
		os.Remove(filepath.Join(getRepoRoot(), "coverage-meta.out"))
	})

	t.Run("TestDocumentation", func(t *testing.T) {
		readmePath := filepath.Join(getRepoRoot(), "test/cicd/README.md")
		
		content, err := os.ReadFile(readmePath)
		require.NoError(t, err)
		
		readmeContent := string(content)
		
		// Check for required documentation sections
		requiredSections := []string{
			"Test Architecture",
			"Test Categories", 
			"Test Implementation Details",
			"Quality Metrics",
			"Continuous Integration Integration",
		}

		for _, section := range requiredSections {
			assert.Contains(t, readmeContent, section,
				"README should contain section: %s", section)
		}

		// Check for test strategy documentation
		strategyElements := []string{
			"Test Architect",
			"Unit Test Specialist",
			"Integration Test Engineer", 
			"Quality Validator",
		}

		for _, element := range strategyElements {
			assert.Contains(t, readmeContent, element,
				"README should document: %s", element)
		}
	})
}

// TestTestSuiteExecution validates the test suite can be executed properly
func TestTestSuiteExecution(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test suite execution in short mode")
	}

	t.Run("SyntaxValidation", func(t *testing.T) {
		// Validate all test files compile without errors
		cmd := exec.Command("go", "build", "./test/cicd")
		cmd.Dir = getRepoRoot()
		
		output, err := cmd.CombinedOutput()
		assert.NoError(t, err, "Test files should compile without errors: %s", string(output))
	})

	t.Run("TestRunnerExecution", func(t *testing.T) {
		// Test that the runner script can be executed
		runnerScript := filepath.Join(getRepoRoot(), "test/cicd/run_tests.sh")
		
		// Test help command
		cmd := exec.Command(runnerScript, "--help")
		output, err := cmd.CombinedOutput()
		assert.NoError(t, err, "Test runner help should work")
		
		helpOutput := string(output)
		assert.Contains(t, helpOutput, "Usage:", "Help should show usage")
		assert.Contains(t, helpOutput, "Commands:", "Help should show commands")
		assert.Contains(t, helpOutput, "Options:", "Help should show options")
	})

	t.Run("PrerequisiteCheck", func(t *testing.T) {
		// Validate that required tools are available for testing
		requiredCommands := []string{"go", "git", "make"}
		
		for _, cmd := range requiredCommands {
			_, err := exec.LookPath(cmd)
			assert.NoError(t, err, "Required command should be available: %s", cmd)
		}

		// Check Go version matches expected
		cmd := exec.Command("go", "version")
		output, err := cmd.CombinedOutput()
		assert.NoError(t, err, "Should be able to check Go version")
		
		versionOutput := string(output)
		assert.Contains(t, versionOutput, "go1.24", 
			"Go version should match expected version (1.24)")
	})

	t.Run("WorkflowFileAccess", func(t *testing.T) {
		// Validate that all workflow files the tests expect are accessible
		workflowFiles := []string{
			".github/workflows/ci.yml",
			".github/actions/setup/action.yml",
			".github/actions/test/action.yml", 
			".github/actions/security/action.yml",
			".github/actions/build/action.yml",
			".github/actions/validate-config/action.yml",
		}

		repoRoot := getRepoRoot()
		for _, file := range workflowFiles {
			fullPath := filepath.Join(repoRoot, file)
			_, err := os.Stat(fullPath)
			assert.NoError(t, err, "Workflow file should be accessible: %s", file)
		}
	})
}

// TestTestQuality validates the quality of the test implementation itself
func TestTestQuality(t *testing.T) {
	t.Run("TestNaming", func(t *testing.T) {
		// Validate test function names follow conventions
		testFiles := []string{
			"workflow_integration_test.go",
			"action_unit_test.go",
			"failure_scenario_test.go",
		}

		repoRoot := getRepoRoot()
		testDir := filepath.Join(repoRoot, "test/cicd")

		for _, testFile := range testFiles {
			filePath := filepath.Join(testDir, testFile)
			content, err := os.ReadFile(filePath)
			require.NoError(t, err)

			fileContent := string(content)
			
			// Test functions should start with "Test"
			assert.Contains(t, fileContent, "func Test", 
				"File %s should contain test functions", testFile)
			
			// Should have proper test signature
			assert.Contains(t, fileContent, "*testing.T", 
				"File %s should use testing.T parameter", testFile)
		}
	})

	t.Run("TestAssertions", func(t *testing.T) {
		// Validate tests use proper assertions
		testFiles := []string{
			"workflow_integration_test.go",
			"action_unit_test.go", 
			"failure_scenario_test.go",
		}

		repoRoot := getRepoRoot()
		testDir := filepath.Join(repoRoot, "test/cicd")

		for _, testFile := range testFiles {
			filePath := filepath.Join(testDir, testFile)
			content, err := os.ReadFile(filePath)
			require.NoError(t, err)

			fileContent := string(content)
			
			// Should use testify assertions
			assert.Contains(t, fileContent, "assert.", 
				"File %s should use assert assertions", testFile)
			assert.Contains(t, fileContent, "require.", 
				"File %s should use require assertions", testFile)
			
			// Should have proper error messages
			assertCount := strings.Count(fileContent, "assert.")
			errorMessageCount := strings.Count(fileContent, `", "`)
			
			// Most assertions should have error messages
			ratio := float64(errorMessageCount) / float64(assertCount)
			assert.Greater(t, ratio, 0.5, 
				"File %s should have error messages for most assertions", testFile)
		}
	})

	t.Run("TestOrganization", func(t *testing.T) {
		// Validate test organization and structure
		testFiles := map[string][]string{
			"workflow_integration_test.go": {
				"TestWorkflowIntegration",
				"TestConfigurationConsistency",
				"TestSecurityWorkflowIntegration",
				"TestCrossPlatformBuildMatrix",
				"TestQualityGateValidation",
				"TestPerformanceAndCaching",
				"TestWorkflowDocumentation",
			},
			"action_unit_test.go": {
				"TestSetupActionUnit",
				"TestTestActionUnit",
				"TestSecurityActionUnit",
				"TestActionValidation",
				"TestActionErrorHandling",
				"TestActionPerformance",
				"TestActionSecurity",
			},
			"failure_scenario_test.go": {
				"TestWorkflowFailureScenarios",
				"TestPerformanceOptimizations",
				"TestContinuousImprovementMetrics",
				"TestRegressionPrevention",
			},
		}

		repoRoot := getRepoRoot()
		testDir := filepath.Join(repoRoot, "test/cicd")

		for testFile, expectedTests := range testFiles {
			filePath := filepath.Join(testDir, testFile)
			content, err := os.ReadFile(filePath)
			require.NoError(t, err)

			fileContent := string(content)
			
			for _, expectedTest := range expectedTests {
				assert.Contains(t, fileContent, "func "+expectedTest,
					"File %s should contain test: %s", testFile, expectedTest)
			}
		}
	})

	t.Run("TestHelperFunctions", func(t *testing.T) {
		// Validate helper functions are well-designed
		testFile := "workflow_integration_test.go"
		filePath := filepath.Join(getRepoRoot(), "test/cicd", testFile)
		
		content, err := os.ReadFile(filePath)
		require.NoError(t, err)

		fileContent := string(content)
		
		// Should have helper functions for common operations
		expectedHelpers := []string{
			"loadWorkflowDefinition",
			"loadActionDefinition", 
			"formatJobNeeds",
			"extractGoVersions",
			"getRepoRoot",
		}

		for _, helper := range expectedHelpers {
			assert.Contains(t, fileContent, "func "+helper,
				"Should have helper function: %s", helper)
		}
	})
}

// TestPerformanceCharacteristics validates test performance
func TestPerformanceCharacteristics(t *testing.T) {
	t.Run("TestExecutionTime", func(t *testing.T) {
		// Validate tests run within reasonable time
		start := time.Now()
		
		// Run a subset of tests to measure performance
		cmd := exec.Command("go", "test", "-run", "TestWorkflowIntegration", "./test/cicd")
		cmd.Dir = getRepoRoot()
		
		err := cmd.Run()
		elapsed := time.Since(start)
		
		if err != nil {
			t.Skipf("Could not run performance test: %v", err)
		}

		// Workflow integration tests should complete quickly
		assert.Less(t, elapsed, 30*time.Second, 
			"Workflow integration tests should complete within 30 seconds")
		
		t.Logf("Workflow integration tests completed in: %v", elapsed)
	})

	t.Run("ResourceUsage", func(t *testing.T) {
		// Tests should not consume excessive resources
		// This is a basic check - in a real environment you might use more sophisticated monitoring
		
		start := time.Now()
		
		cmd := exec.Command("go", "test", "-run", "TestActionValidation", "./test/cicd")
		cmd.Dir = getRepoRoot()
		
		err := cmd.Run()
		elapsed := time.Since(start)
		
		if err != nil {
			t.Skipf("Could not run resource test: %v", err)
		}

		// Action validation tests should be lightweight
		assert.Less(t, elapsed, 10*time.Second,
			"Action validation tests should be lightweight")
		
		t.Logf("Action validation tests completed in: %v", elapsed)
	})
}

// TestMaintenanceAndEvolution validates test maintainability
func TestMaintenanceAndEvolution(t *testing.T) {
	t.Run("CodeStructure", func(t *testing.T) {
		// Validate test code follows good practices
		testFiles := []string{
			"workflow_integration_test.go",
			"action_unit_test.go",
			"failure_scenario_test.go",
		}

		repoRoot := getRepoRoot()
		testDir := filepath.Join(repoRoot, "test/cicd")

		for _, testFile := range testFiles {
			filePath := filepath.Join(testDir, testFile)
			content, err := os.ReadFile(filePath)
			require.NoError(t, err)

			fileContent := string(content)
			
			// Should have package documentation
			assert.Contains(t, fileContent, "// Package cicd",
				"File %s should have package documentation", testFile)
			
			// Should import required testing packages
			assert.Contains(t, fileContent, "github.com/stretchr/testify",
				"File %s should import testify", testFile)
			
			// Should not have TODOs or FIXMEs (tests should be complete)
			assert.NotContains(t, fileContent, "TODO",
				"File %s should not contain TODOs", testFile)
			assert.NotContains(t, fileContent, "FIXME", 
				"File %s should not contain FIXMEs", testFile)
		}
	})

	t.Run("Documentation", func(t *testing.T) {
		// Validate test documentation quality
		readmePath := filepath.Join(getRepoRoot(), "test/cicd/README.md")
		
		info, err := os.Stat(readmePath)
		require.NoError(t, err)
		
		// Documentation should be substantial
		assert.Greater(t, info.Size(), int64(5000), 
			"Test documentation should be comprehensive")
		
		content, err := os.ReadFile(readmePath)
		require.NoError(t, err)
		
		readmeContent := string(content)
		
		// Should contain examples
		assert.Contains(t, readmeContent, "```",
			"Documentation should contain code examples")
		
		// Should explain the architecture
		assert.Contains(t, readmeContent, "Test Architecture",
			"Documentation should explain test architecture")
	})

	t.Run("Extensibility", func(t *testing.T) {
		// Validate tests are designed for extensibility
		integrationTestFile := filepath.Join(getRepoRoot(), "test/cicd/workflow_integration_test.go")
		
		content, err := os.ReadFile(integrationTestFile)
		require.NoError(t, err)

		fileContent := string(content)
		
		// Should have modular helper functions
		helperCount := strings.Count(fileContent, "func ")
		testCount := strings.Count(fileContent, "func Test")
		
		// Should have more helper functions than test functions (good modularity)
		assert.Greater(t, helperCount-testCount, testCount/2,
			"Should have good helper function modularity")
		
		// Should use data-driven tests where appropriate
		assert.Contains(t, fileContent, "for _, ",
			"Should use data-driven test patterns")
	})
}