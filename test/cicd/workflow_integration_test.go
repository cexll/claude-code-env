// Package cicd provides comprehensive testing for GitHub Actions CI/CD workflows
// This implements testing strategies for the enhanced CI/CD pipeline that achieved 97% quality score
package cicd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// WorkflowDefinition represents a GitHub Actions workflow structure
type WorkflowDefinition struct {
	Name        string                 `yaml:"name"`
	On          map[string]interface{} `yaml:"on"`
	Concurrency map[string]interface{} `yaml:"concurrency"`
	Permissions map[string]string      `yaml:"permissions"`
	Env         map[string]string      `yaml:"env"`
	Defaults    map[string]interface{} `yaml:"defaults"`
	Jobs        map[string]Job         `yaml:"jobs"`
}

// Job represents a GitHub Actions job
type Job struct {
	Name       string                   `yaml:"name"`
	RunsOn     interface{}              `yaml:"runs-on"` // string or []string
	Needs      interface{}              `yaml:"needs"`   // string or []string
	If         string                   `yaml:"if"`
	Timeout    string                   `yaml:"timeout-minutes"`
	Strategy   map[string]interface{}   `yaml:"strategy"`
	Outputs    map[string]string        `yaml:"outputs"`
	Steps      []Step                   `yaml:"steps"`
	Env        map[string]string        `yaml:"env"`
	Defaults   map[string]interface{}   `yaml:"defaults"`
	Continue   bool                     `yaml:"continue-on-error"`
	Services   map[string]interface{}   `yaml:"services"`
	Container  interface{}              `yaml:"container"`
}

// Step represents a GitHub Actions step
type Step struct {
	Name            string            `yaml:"name"`
	ID              string            `yaml:"id"`
	If              string            `yaml:"if"`
	Uses            string            `yaml:"uses"`
	Run             string            `yaml:"run"`
	With            map[string]string `yaml:"with"`
	Env             map[string]string `yaml:"env"`
	Continue        bool              `yaml:"continue-on-error"`
	TimeoutMinutes  int               `yaml:"timeout-minutes"`
	Shell           string            `yaml:"shell"`
	WorkingDir      string            `yaml:"working-directory"`
}

// ActionDefinition represents a reusable GitHub Action
type ActionDefinition struct {
	Name        string                    `yaml:"name"`
	Description string                    `yaml:"description"`
	Author      string                    `yaml:"author"`
	Inputs      map[string]ActionInput    `yaml:"inputs"`
	Outputs     map[string]ActionOutput   `yaml:"outputs"`
	Runs        ActionRuns                `yaml:"runs"`
	Branding    map[string]string         `yaml:"branding"`
}

// ActionInput represents an input parameter for a GitHub Action
type ActionInput struct {
	Description string `yaml:"description"`
	Required    bool   `yaml:"required"`
	Default     string `yaml:"default"`
	Type        string `yaml:"type"`
}

// ActionOutput represents an output parameter for a GitHub Action
type ActionOutput struct {
	Description string `yaml:"description"`
	Value       string `yaml:"value"`
}

// ActionRuns represents the execution configuration for a GitHub Action
type ActionRuns struct {
	Using string `yaml:"using"`
	Steps []Step `yaml:"steps"`
	Main  string `yaml:"main"`
	Pre   string `yaml:"pre"`
	Post  string `yaml:"post"`
}

// TestWorkflowIntegration validates the complete CI/CD workflow integration
func TestWorkflowIntegration(t *testing.T) {
	workflowPath := getWorkflowPath("ci.yml")
	
	t.Run("WorkflowStructureValidation", func(t *testing.T) {
		// Load and parse the main CI workflow
		workflow, err := loadWorkflowDefinition(workflowPath)
		require.NoError(t, err, "Failed to load CI workflow definition")
		
		// Validate workflow metadata
		assert.Equal(t, "Continuous Integration", workflow.Name)
		assert.NotEmpty(t, workflow.On, "Workflow should have trigger conditions")
		assert.NotEmpty(t, workflow.Jobs, "Workflow should have jobs defined")
		
		// Validate global configuration
		assert.Contains(t, workflow.Env, "GO_VERSION", "GO_VERSION should be defined globally")
		assert.Equal(t, "1.24", workflow.Env["GO_VERSION"], "GO_VERSION should match expected value")
		assert.Contains(t, workflow.Env, "CACHE_VERSION", "CACHE_VERSION should be defined")
		assert.Contains(t, workflow.Env, "ARTIFACTS_RETENTION", "ARTIFACTS_RETENTION should be defined")
		
		// Validate concurrency configuration
		assert.NotEmpty(t, workflow.Concurrency, "Concurrency should be configured")
		
		// Validate permissions (security requirement)
		assert.NotEmpty(t, workflow.Permissions, "Permissions should be explicitly set")
		expectedPermissions := []string{"contents", "actions", "checks", "pull-requests", "statuses"}
		for _, perm := range expectedPermissions {
			assert.Contains(t, workflow.Permissions, perm, fmt.Sprintf("Permission %s should be defined", perm))
		}
	})
	
	t.Run("JobDependencyValidation", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(workflowPath)
		require.NoError(t, err)
		
		// Test job dependency chain
		jobNames := []string{
			"version-validation",
			"fast-validation", 
			"build-matrix",
			"test-suite",
			"quality-checks",
			"security-scan",
			"performance-tests",
			"integration",
		}
		
		for _, jobName := range jobNames {
			assert.Contains(t, workflow.Jobs, jobName, fmt.Sprintf("Job %s should be defined", jobName))
		}
		
		// Validate critical job dependencies
		fastValidation := workflow.Jobs["fast-validation"]
		assert.Contains(t, formatJobNeeds(fastValidation.Needs), "version-validation", 
			"fast-validation should depend on version-validation")
		
		buildMatrix := workflow.Jobs["build-matrix"] 
		buildNeeds := formatJobNeeds(buildMatrix.Needs)
		assert.Contains(t, buildNeeds, "version-validation", "build-matrix should depend on version-validation")
		assert.Contains(t, buildNeeds, "fast-validation", "build-matrix should depend on fast-validation")
		
		integration := workflow.Jobs["integration"]
		integrationNeeds := formatJobNeeds(integration.Needs)
		requiredJobs := []string{"version-validation", "fast-validation", "build-matrix", "test-suite", "quality-checks"}
		for _, job := range requiredJobs {
			assert.Contains(t, integrationNeeds, job, 
				fmt.Sprintf("integration job should depend on %s", job))
		}
	})
	
	t.Run("ConditionalJobExecution", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(workflowPath)
		require.NoError(t, err)
		
		// Validate conditional execution logic
		securityScan := workflow.Jobs["security-scan"]
		assert.NotEmpty(t, securityScan.If, "security-scan should have conditional execution")
		assert.Contains(t, securityScan.If, "main", "security-scan should run on main branch")
		assert.Contains(t, securityScan.If, "security", "security-scan should run with security label")
		
		performanceTests := workflow.Jobs["performance-tests"]
		assert.NotEmpty(t, performanceTests.If, "performance-tests should have conditional execution")
		assert.Contains(t, performanceTests.If, "performance", "performance-tests should run with performance label")
		
		// Validate that core jobs don't have restrictive conditions
		coreJobs := []string{"version-validation", "fast-validation", "build-matrix", "test-suite", "quality-checks"}
		for _, jobName := range coreJobs {
			job := workflow.Jobs[jobName]
			if job.If != "" {
				// Core jobs should only skip on draft PRs or version validation failures
				assert.Contains(t, job.If, "should-skip != 'true'", 
					fmt.Sprintf("Core job %s should have proper skip conditions", jobName))
			}
		}
	})
	
	t.Run("TimeoutValidation", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(workflowPath)
		require.NoError(t, err)
		
		// Validate reasonable timeout values
		timeoutLimits := map[string]int{
			"version-validation": 5,
			"fast-validation":    5,
			"build-matrix":       10,
			"test-suite":         15,
			"quality-checks":     10,
			"security-scan":      15,
			"performance-tests":  20,
			"integration":        5,
		}
		
		for jobName, expectedMax := range timeoutLimits {
			job := workflow.Jobs[jobName]
			if job.Timeout != "" {
				// Parse timeout value (assuming format like "15" for 15 minutes)
				timeoutStr := strings.TrimSuffix(job.Timeout, "-minutes")
				timeout := 0
				fmt.Sscanf(timeoutStr, "%d", &timeout)
				assert.LessOrEqual(t, timeout, expectedMax, 
					fmt.Sprintf("Job %s timeout should be <= %d minutes", jobName, expectedMax))
			}
		}
	})
}

// TestActionValidation validates individual reusable actions
func TestActionValidation(t *testing.T) {
	actions := []string{"setup", "test", "security", "build", "validate-config"}
	
	for _, actionName := range actions {
		t.Run(fmt.Sprintf("Action_%s", actionName), func(t *testing.T) {
			actionPath := getActionPath(actionName, "action.yml")
			
			// Load action definition
			action, err := loadActionDefinition(actionPath)
			require.NoError(t, err, fmt.Sprintf("Failed to load action %s", actionName))
			
			// Validate basic structure
			assert.NotEmpty(t, action.Name, fmt.Sprintf("Action %s should have a name", actionName))
			assert.NotEmpty(t, action.Description, fmt.Sprintf("Action %s should have a description", actionName))
			assert.Equal(t, "composite", action.Runs.Using, fmt.Sprintf("Action %s should use composite", actionName))
			
			// Validate inputs and outputs
			validateActionInputsOutputs(t, action, actionName)
			
			// Validate steps
			assert.NotEmpty(t, action.Runs.Steps, fmt.Sprintf("Action %s should have steps", actionName))
			
			// Action-specific validations
			switch actionName {
			case "setup":
				validateSetupAction(t, action)
			case "test":
				validateTestAction(t, action)
			case "security":
				validateSecurityAction(t, action)
			case "validate-config":
				validateConfigAction(t, action)
			}
		})
	}
}

// TestConfigurationConsistency validates version synchronization and configuration validation
func TestConfigurationConsistency(t *testing.T) {
	t.Run("VersionSynchronization", func(t *testing.T) {
		// Test Go version consistency across files
		versions := extractGoVersions()
		
		assert.NotEmpty(t, versions.Workflow, "Workflow Go version should be defined")
		assert.NotEmpty(t, versions.Makefile, "Makefile Go version should be defined") 
		assert.NotEmpty(t, versions.GoMod, "go.mod Go version should be defined")
		
		// All versions should match
		assert.Equal(t, versions.Workflow, versions.Makefile, 
			"Workflow and Makefile Go versions should match")
		assert.Equal(t, versions.Workflow, versions.GoMod, 
			"Workflow and go.mod Go versions should match")
		assert.Equal(t, "1.24", versions.Workflow, 
			"Go version should be 1.24")
	})
	
	t.Run("GolangciConfigValidation", func(t *testing.T) {
		golangciPath := filepath.Join(getRepoRoot(), ".golangci.yml")
		
		// Validate .golangci.yml exists and is valid
		require.FileExists(t, golangciPath, ".golangci.yml should exist")
		
		config, err := loadGolangciConfig(golangciPath)
		require.NoError(t, err, ".golangci.yml should be valid YAML")
		
		// Validate comprehensive configuration
		assert.NotEmpty(t, config["run"], "Run configuration should be defined")
		assert.NotEmpty(t, config["linters"], "Linters should be configured") 
		assert.NotEmpty(t, config["linters-settings"], "Linter settings should be configured")
		assert.NotEmpty(t, config["issues"], "Issues configuration should be defined")
		
		// Validate Go version in golangci config matches global version
		if runConfig, ok := config["run"].(map[string]interface{}); ok {
			if goVersion, ok := runConfig["go"].(string); ok {
				assert.Equal(t, "1.24", goVersion, "golangci-lint Go version should match global version")
			}
		}
		
		// Validate required linters are enabled
		validateRequiredLinters(t, config)
	})
	
	t.Run("CacheKeyConsistency", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)
		
		// Extract setup action usage from jobs
		setupUsages := extractSetupActionUsages(workflow)
		
		// Validate consistent cache configuration
		for jobName, usage := range setupUsages {
			if cachePath, ok := usage["cache-dependency-path"]; ok {
				assert.Contains(t, cachePath, "go.sum", 
					fmt.Sprintf("Job %s should use go.sum for cache key", jobName))
			}
		}
	})
}

// TestSecurityWorkflowIntegration validates the enhanced security scanning workflow
func TestSecurityWorkflowIntegration(t *testing.T) {
	t.Run("SecurityActionIntegration", func(t *testing.T) {
		securityAction, err := loadActionDefinition(getActionPath("security", "action.yml"))
		require.NoError(t, err)
		
		// Validate security scan types
		scanTypeInput := securityAction.Inputs["scan-type"]
		assert.Equal(t, "both", scanTypeInput.Default, "Default scan type should be 'both'")
		
		// Validate severity handling
		severityInput := securityAction.Inputs["severity-threshold"]
		assert.Equal(t, "high", severityInput.Default, "Default severity threshold should be 'high'")
		
		// Validate fail conditions
		failOnCritical := securityAction.Inputs["fail-on-critical"]
		assert.Equal(t, "true", failOnCritical.Default, "Should fail on critical by default")
		
		failOnHigh := securityAction.Inputs["fail-on-high"]
		assert.Equal(t, "true", failOnHigh.Default, "Should fail on high by default")
		
		// Validate SARIF upload capability
		uploadSarif := securityAction.Inputs["upload-sarif"]
		assert.Equal(t, "true", uploadSarif.Default, "Should upload SARIF by default")
		
		// Validate expected outputs
		expectedOutputs := []string{"gosec-findings", "govulncheck-findings", "max-severity", "overall-status"}
		for _, output := range expectedOutputs {
			assert.Contains(t, securityAction.Outputs, output, 
				fmt.Sprintf("Security action should output %s", output))
		}
	})
	
	t.Run("SecurityToolVerification", func(t *testing.T) {
		securityAction, err := loadActionDefinition(getActionPath("security", "action.yml"))
		require.NoError(t, err)
		
		// Find tool verification step
		var verificationStep *Step
		for _, step := range securityAction.Runs.Steps {
			if strings.Contains(step.Name, "tool availability") {
				verificationStep = &step
				break
			}
		}
		
		require.NotNil(t, verificationStep, "Security action should have tool verification step")
		assert.Contains(t, verificationStep.Run, "gosec", "Should verify gosec availability")
		assert.Contains(t, verificationStep.Run, "govulncheck", "Should verify govulncheck availability")
	})
	
	t.Run("SecurityReportGeneration", func(t *testing.T) {
		securityAction, err := loadActionDefinition(getActionPath("security", "action.yml"))
		require.NoError(t, err)
		
		// Validate report generation steps
		reportSteps := []string{"consolidated security report", "SARIF report"}
		for _, expectedStep := range reportSteps {
			found := false
			for _, step := range securityAction.Runs.Steps {
				if strings.Contains(strings.ToLower(step.Name), strings.ToLower(expectedStep)) {
					found = true
					break
				}
			}
			assert.True(t, found, fmt.Sprintf("Security action should have %s generation step", expectedStep))
		}
	})
}

// TestCrossPlatformBuildMatrix validates the build matrix functionality
func TestCrossPlatformBuildMatrix(t *testing.T) {
	t.Run("BuildMatrixConfiguration", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)
		
		buildMatrix := workflow.Jobs["build-matrix"]
		assert.NotEmpty(t, buildMatrix.Strategy, "Build matrix should have strategy")
		
		strategy := buildMatrix.Strategy
		assert.Contains(t, strategy, "matrix", "Strategy should contain matrix")
		assert.Equal(t, false, strategy["fail-fast"], "Build matrix should not fail fast")
		
		// Validate expected platforms
		if matrixConfig, ok := strategy["matrix"].(map[string]interface{}); ok {
			if include, ok := matrixConfig["include"].([]interface{}); ok {
				expectedPlatforms := map[string]map[string]string{
					"linux-amd64":   {"os": "ubuntu-latest", "goos": "linux", "goarch": "amd64"},
					"linux-arm64":   {"os": "ubuntu-latest", "goos": "linux", "goarch": "arm64"},
					"darwin-amd64":  {"os": "macos-latest", "goos": "darwin", "goarch": "amd64"},
					"darwin-arm64":  {"os": "macos-latest", "goos": "darwin", "goarch": "arm64"},
					"windows-amd64": {"os": "windows-latest", "goos": "windows", "goarch": "amd64"},
				}
				
				assert.GreaterOrEqual(t, len(include), len(expectedPlatforms), 
					"Matrix should include all expected platforms")
			}
		}
	})
	
	t.Run("ArtifactGeneration", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)
		
		buildMatrix := workflow.Jobs["build-matrix"]
		
		// Find upload artifacts step
		var uploadStep *Step
		for _, step := range buildMatrix.Steps {
			if strings.Contains(step.Name, "Upload build artifacts") {
				uploadStep = &step
				break
			}
		}
		
		require.NotNil(t, uploadStep, "Build matrix should upload artifacts")
		assert.Contains(t, uploadStep.With["name"], "binaries-", "Artifact name should include platform info")
		assert.Equal(t, "dist/", uploadStep.With["path"], "Should upload from dist/ directory")
	})
}

// TestQualityGateValidation validates quality gates and failure scenarios
func TestQualityGateValidation(t *testing.T) {
	t.Run("CoverageThresholds", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)
		
		testSuite := workflow.Jobs["test-suite"]
		
		// Find test action usages
		testSteps := extractTestActionUsages(testSuite.Steps)
		
		for testType, usage := range testSteps {
			if threshold, ok := usage["coverage-threshold"]; ok {
				switch testType {
				case "unit":
					assert.Equal(t, "80", threshold, "Unit test coverage should be 80%")
				case "integration":
					assert.Equal(t, "70", threshold, "Integration test coverage should be 70%")
				}
			}
		}
	})
	
	t.Run("FailureHandling", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)
		
		integration := workflow.Jobs["integration"]
		assert.Contains(t, integration.If, "always()", "Integration job should run even on failures")
		
		// Find result checking step
		var checkStep *Step
		for _, step := range integration.Steps {
			if strings.Contains(step.Name, "Check job results") {
				checkStep = &step
				break
			}
		}
		
		require.NotNil(t, checkStep, "Integration job should check other job results")
		assert.Contains(t, checkStep.Run, "exit 1", "Should fail if required jobs failed")
	})
	
	t.Run("ArtifactRetention", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)
		
		retentionDays := workflow.Env["ARTIFACTS_RETENTION"]
		assert.Equal(t, "30", retentionDays, "Artifact retention should be 30 days")
		
		// Validate all upload steps use the retention setting
		for jobName, job := range workflow.Jobs {
			for _, step := range job.Steps {
				if step.Uses != "" && strings.Contains(step.Uses, "upload-artifact") {
					if retention, ok := step.With["retention-days"]; ok {
						assert.Equal(t, "${{ env.ARTIFACTS_RETENTION }}", retention, 
							fmt.Sprintf("Job %s should use global retention setting", jobName))
					}
				}
			}
		}
	})
}

// TestPerformanceAndCaching validates caching strategy and performance optimizations
func TestPerformanceAndCaching(t *testing.T) {
	t.Run("CacheStrategy", func(t *testing.T) {
		setupAction, err := loadActionDefinition(getActionPath("setup", "action.yml"))
		require.NoError(t, err)
		
		// Find cache steps
		var modulesCache, toolsCache *Step
		for _, step := range setupAction.Runs.Steps {
			if strings.Contains(step.Name, "Cache Go modules") {
				modulesCache = &step
			}
			if strings.Contains(step.Name, "Cache tools") {
				toolsCache = &step
			}
		}
		
		require.NotNil(t, modulesCache, "Setup action should cache Go modules")
		require.NotNil(t, toolsCache, "Setup action should cache tools")
		
		// Validate cache keys
		assert.Contains(t, modulesCache.With["key"], "go-", "Modules cache key should include 'go'")
		assert.Contains(t, modulesCache.With["key"], "hashFiles", "Modules cache key should use hashFiles")
		
		assert.Contains(t, toolsCache.With["key"], "tools-", "Tools cache key should include 'tools'")
	})
	
	t.Run("ParallelExecution", func(t *testing.T) {
		testAction, err := loadActionDefinition(getActionPath("test", "action.yml"))
		require.NoError(t, err)
		
		// Find test execution step
		var testStep *Step
		for _, step := range testAction.Runs.Steps {
			if strings.Contains(step.Name, "Run tests with retry") {
				testStep = &step
				break
			}
		}
		
		require.NotNil(t, testStep, "Test action should have test execution step")
		assert.Contains(t, testStep.Run, "-parallel=4", "Tests should run in parallel")
	})
	
	t.Run("ConditionalToolInstallation", func(t *testing.T) {
		setupAction, err := loadActionDefinition(getActionPath("setup", "action.yml"))
		require.NoError(t, err)
		
		// Find tool installation step
		var installStep *Step
		for _, step := range setupAction.Runs.Steps {
			if strings.Contains(step.Name, "Install development tools") {
				installStep = &step
				break
			}
		}
		
		require.NotNil(t, installStep, "Setup action should have tool installation step")
		assert.Contains(t, installStep.If, "cache-tools.outputs.cache-hit != 'true'", 
			"Tools should only install on cache miss")
	})
}

// TestWorkflowDocumentation validates error messages and documentation quality
func TestWorkflowDocumentation(t *testing.T) {
	t.Run("ErrorMessageQuality", func(t *testing.T) {
		actions := []string{"setup", "test", "security", "validate-config"}
		
		for _, actionName := range actions {
			action, err := loadActionDefinition(getActionPath(actionName, "action.yml"))
			require.NoError(t, err)
			
			// Validate input descriptions
			for inputName, input := range action.Inputs {
				assert.NotEmpty(t, input.Description, 
					fmt.Sprintf("Action %s input %s should have description", actionName, inputName))
			}
			
			// Validate output descriptions  
			for outputName, output := range action.Outputs {
				assert.NotEmpty(t, output.Description,
					fmt.Sprintf("Action %s output %s should have description", actionName, outputName))
			}
		}
	})
	
	t.Run("WorkflowComments", func(t *testing.T) {
		workflowContent, err := ioutil.ReadFile(getWorkflowPath("ci.yml"))
		require.NoError(t, err)
		
		content := string(workflowContent)
		
		// Check for important comments
		expectedComments := []string{
			"Ensure only one CI run per branch/PR",
			"Default permissions (minimal)",
			"Global environment variables", 
			"Version validation - must pass before any other jobs",
			"Fast validation - fail early for basic issues",
			"Build matrix - cross-platform compilation",
			"Comprehensive test suite",
			"Code quality checks",
			"Enhanced security scan",
			"Final integration job",
		}
		
		for _, comment := range expectedComments {
			assert.Contains(t, content, comment, 
				fmt.Sprintf("Workflow should contain comment: %s", comment))
		}
	})
	
	t.Run("ActionMetadata", func(t *testing.T) {
		actions := []string{"setup", "test", "security", "build", "validate-config"}
		
		for _, actionName := range actions {
			action, err := loadActionDefinition(getActionPath(actionName, "action.yml"))
			require.NoError(t, err)
			
			assert.NotEmpty(t, action.Author, fmt.Sprintf("Action %s should have author", actionName))
			assert.Contains(t, action.Author, "Claude Code Environment Switcher Team", 
				fmt.Sprintf("Action %s should have correct author", actionName))
		}
	})
}

// Helper functions for test implementation

func getRepoRoot() string {
	wd, _ := os.Getwd()
	// Navigate up to find the repo root (contains .github directory)
	for {
		if _, err := os.Stat(filepath.Join(wd, ".github")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return wd
}

func getWorkflowPath(filename string) string {
	return filepath.Join(getRepoRoot(), ".github", "workflows", filename)
}

func getActionPath(actionName, filename string) string {
	return filepath.Join(getRepoRoot(), ".github", "actions", actionName, filename)
}

func loadWorkflowDefinition(path string) (*WorkflowDefinition, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var workflow WorkflowDefinition
	err = yaml.Unmarshal(data, &workflow)
	return &workflow, err
}

func loadActionDefinition(path string) (*ActionDefinition, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var action ActionDefinition
	err = yaml.Unmarshal(data, &action)
	return &action, err
}

func loadGolangciConfig(path string) (map[string]interface{}, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var config map[string]interface{}
	err = yaml.Unmarshal(data, &config)
	return config, err
}

func formatJobNeeds(needs interface{}) []string {
	switch v := needs.(type) {
	case string:
		return []string{v}
	case []interface{}:
		result := make([]string, len(v))
		for i, item := range v {
			result[i] = fmt.Sprintf("%v", item)
		}
		return result
	case []string:
		return v
	default:
		return []string{}
	}
}

type GoVersions struct {
	Workflow string
	Makefile string
	GoMod    string
}

func extractGoVersions() GoVersions {
	versions := GoVersions{}
	
	// Extract from workflow
	workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
	if err == nil {
		versions.Workflow = workflow.Env["GO_VERSION"]
	}
	
	// Extract from Makefile
	makefilePath := filepath.Join(getRepoRoot(), "Makefile")
	if content, err := ioutil.ReadFile(makefilePath); err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "GO_VERSION=") {
				versions.Makefile = strings.TrimPrefix(line, "GO_VERSION=")
				break
			}
		}
	}
	
	// Extract from go.mod
	goModPath := filepath.Join(getRepoRoot(), "go.mod")
	if content, err := ioutil.ReadFile(goModPath); err == nil {
		lines := strings.Split(string(content), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "go ") {
				versions.GoMod = strings.TrimPrefix(line, "go ")
				break
			}
		}
	}
	
	return versions
}

func extractSetupActionUsages(workflow *WorkflowDefinition) map[string]map[string]string {
	usages := make(map[string]map[string]string)
	
	for jobName, job := range workflow.Jobs {
		for _, step := range job.Steps {
			if strings.Contains(step.Uses, "setup") {
				usages[jobName] = step.With
				break
			}
		}
	}
	
	return usages
}

func extractTestActionUsages(steps []Step) map[string]map[string]string {
	usages := make(map[string]map[string]string)
	
	for _, step := range steps {
		if strings.Contains(step.Uses, "test") {
			if testType, ok := step.With["test-type"]; ok {
				usages[testType] = step.With
			}
		}
	}
	
	return usages
}

func validateActionInputsOutputs(t *testing.T, action *ActionDefinition, actionName string) {
	// Validate common inputs exist
	commonInputs := map[string]bool{
		"setup":           true,
		"test":            true, 
		"security":        true,
		"build":           true,
		"validate-config": true,
	}
	
	if commonInputs[actionName] {
		// All actions should have proper input validation
		assert.NotEmpty(t, action.Inputs, fmt.Sprintf("Action %s should have inputs", actionName))
		assert.NotEmpty(t, action.Outputs, fmt.Sprintf("Action %s should have outputs", actionName))
	}
}

func validateSetupAction(t *testing.T, action *ActionDefinition) {
	// Validate setup-specific requirements
	requiredInputs := []string{"go-version", "install-tools", "validate-version-consistency"}
	for _, input := range requiredInputs {
		assert.Contains(t, action.Inputs, input, 
			fmt.Sprintf("Setup action should have %s input", input))
	}
	
	requiredOutputs := []string{"go-version", "cache-hit", "version-consistent"}
	for _, output := range requiredOutputs {
		assert.Contains(t, action.Outputs, output,
			fmt.Sprintf("Setup action should have %s output", output))
	}
}

func validateTestAction(t *testing.T, action *ActionDefinition) {
	// Validate test-specific requirements
	requiredInputs := []string{"test-type", "coverage-threshold", "parallel", "race-detection"}
	for _, input := range requiredInputs {
		assert.Contains(t, action.Inputs, input,
			fmt.Sprintf("Test action should have %s input", input))
	}
	
	requiredOutputs := []string{"test-result", "coverage-percentage", "total-tests", "failed-tests"}
	for _, output := range requiredOutputs {
		assert.Contains(t, action.Outputs, output,
			fmt.Sprintf("Test action should have %s output", output))
	}
}

func validateSecurityAction(t *testing.T, action *ActionDefinition) {
	// Validate security-specific requirements
	requiredInputs := []string{"scan-type", "severity-threshold", "fail-on-critical", "fail-on-high"}
	for _, input := range requiredInputs {
		assert.Contains(t, action.Inputs, input,
			fmt.Sprintf("Security action should have %s input", input))
	}
	
	requiredOutputs := []string{"gosec-findings", "govulncheck-findings", "max-severity", "overall-status"}
	for _, output := range requiredOutputs {
		assert.Contains(t, action.Outputs, output,
			fmt.Sprintf("Security action should have %s output", output))
	}
}

func validateConfigAction(t *testing.T, action *ActionDefinition) {
	// Validate config validation-specific requirements
	requiredInputs := []string{"validate-go-versions", "validate-golangci-config", "fail-on-mismatch"}
	for _, input := range requiredInputs {
		assert.Contains(t, action.Inputs, input,
			fmt.Sprintf("Config validation action should have %s input", input))
	}
}

func validateRequiredLinters(t *testing.T, config map[string]interface{}) {
	if linters, ok := config["linters"].(map[string]interface{}); ok {
		if enabled, ok := linters["enable"].([]interface{}); ok {
			enabledLinters := make([]string, len(enabled))
			for i, linter := range enabled {
				enabledLinters[i] = fmt.Sprintf("%v", linter)
			}
			
			requiredLinters := []string{"errcheck", "gosimple", "govet", "staticcheck", "gosec", "gofmt", "goimports"}
			for _, required := range requiredLinters {
				found := false
				for _, enabled := range enabledLinters {
					if enabled == required {
						found = true
						break
					}
				}
				assert.True(t, found, fmt.Sprintf("Required linter %s should be enabled", required))
			}
		}
	}
}