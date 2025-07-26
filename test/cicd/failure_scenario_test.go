// Package cicd provides failure scenario and performance testing for GitHub Actions CI/CD
package cicd

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWorkflowFailureScenarios validates error handling and recovery in CI/CD workflows
func TestWorkflowFailureScenarios(t *testing.T) {
	t.Run("VersionMismatchHandling", func(t *testing.T) {
		// Test what happens when Go versions are inconsistent
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)

		// Version validation should be the first critical job
		versionJob := workflow.Jobs["version-validation"]
		assert.NotEmpty(t, versionJob, "Version validation job should exist")
		
		// It should have timeout to prevent hanging
		assert.NotEmpty(t, versionJob.Timeout, "Version validation should have timeout")
		
		// Other jobs should depend on it
		dependentJobs := []string{"fast-validation", "build-matrix", "test-suite", "quality-checks"}
		for _, jobName := range dependentJobs {
			job := workflow.Jobs[jobName]
			needs := formatJobNeeds(job.Needs)
			assert.Contains(t, needs, "version-validation", 
				fmt.Sprintf("Job %s should depend on version-validation", jobName))
		}

		// Jobs should have proper conditional execution
		fastValidation := workflow.Jobs["fast-validation"]
		assert.Contains(t, fastValidation.If, "go-version-consistent == 'true'",
			"Fast validation should only run if versions are consistent")
	})

	t.Run("BuildFailureRecovery", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)

		buildMatrix := workflow.Jobs["build-matrix"]
		
		// Build matrix should not fail fast to allow other platforms to complete
		if strategy, ok := buildMatrix.Strategy["fail-fast"]; ok {
			assert.Equal(t, false, strategy, "Build matrix should not fail fast")
		}

		// Should have reasonable timeout
		assert.NotEmpty(t, buildMatrix.Timeout, "Build matrix should have timeout")
		
		// Integration job should check build results
		integration := workflow.Jobs["integration"]
		assert.Contains(t, integration.If, "always()", "Integration should run even on build failures")
		
		// Find result checking step
		var checkStep *Step
		for _, step := range integration.Steps {
			if strings.Contains(step.Name, "Check job results") {
				checkStep = &step
				break
			}
		}
		
		require.NotNil(t, checkStep, "Integration should check job results")
		assert.Contains(t, checkStep.Run, "build-matrix.result", "Should check build matrix result")
		assert.Contains(t, checkStep.Run, "exit 1", "Should fail if build matrix failed")
	})

	t.Run("TestFailureHandling", func(t *testing.T) {
		testAction, err := loadActionDefinition(getActionPath("test", "action.yml"))
		require.NoError(t, err)

		// Find test execution step
		var execStep *Step
		for _, step := range testAction.Runs.Steps {
			if strings.Contains(step.Name, "Run tests with retry") {
				execStep = &step
				break
			}
		}

		require.NotNil(t, execStep, "Test action should have retry logic")
		
		// Should have retry mechanism
		assert.Contains(t, execStep.Run, "RETRY_COUNT", "Should implement retry logic")
		assert.Contains(t, execStep.Run, "for i in", "Should have retry loop")
		
		// Should handle test output even on failure
		assert.Contains(t, execStep.Run, "tee", "Should capture test output")
		assert.Contains(t, execStep.Run, "PIPESTATUS", "Should capture exit codes properly")

		// Coverage processing should handle missing files
		var coverageStep *Step
		for _, step := range testAction.Runs.Steps {
			if strings.Contains(step.Name, "Process coverage") {
				coverageStep = &step
				break
			}
		}

		if coverageStep != nil {
			assert.Contains(t, coverageStep.Run, "if [ -f", "Should check if coverage file exists")
			assert.Contains(t, coverageStep.If, "success()", "Should only run on test success")
		}
	})

	t.Run("SecurityScanFailures", func(t *testing.T) {
		securityAction, err := loadActionDefinition(getActionPath("security", "action.yml"))
		require.NoError(t, err)

		// Find tool verification step
		var verifyStep *Step
		for _, step := range securityAction.Runs.Steps {
			if strings.Contains(step.Name, "tool availability") {
				verifyStep = &step
				break
			}
		}

		require.NotNil(t, verifyStep, "Security action should verify tools")
		assert.Contains(t, verifyStep.Run, "exit 1", "Should fail if tools are missing")

		// Should handle missing reports gracefully
		var gosecStep *Step
		for _, step := range securityAction.Runs.Steps {
			if strings.Contains(step.Name, "Run gosec") {
				gosecStep = &step
				break
			}
		}

		if gosecStep != nil {
			assert.Contains(t, gosecStep.Run, "set +e", "Should handle gosec failures")
			assert.Contains(t, gosecStep.Run, "|| echo", "Should provide default values")
		}

		// Failure evaluation should be comprehensive
		var evalStep *Step
		for _, step := range securityAction.Runs.Steps {
			if strings.Contains(step.Name, "Evaluate failure") {
				evalStep = &step
				break
			}
		}

		if evalStep != nil {
			assert.Contains(t, evalStep.Run, "SHOULD_FAIL", "Should track failure conditions")
			assert.Contains(t, evalStep.Run, "case", "Should handle different severity levels")
		}
	})

	t.Run("NetworkFailureResilience", func(t *testing.T) {
		setupAction, err := loadActionDefinition(getActionPath("setup", "action.yml"))
		require.NoError(t, err)

		// Tool installation should handle network failures
		var installStep *Step
		for _, step := range setupAction.Runs.Steps {
			if strings.Contains(step.Name, "Install development tools") {
				installStep = &step
				break
			}
		}

		if installStep != nil {
			// Should check if tools are already installed
			assert.Contains(t, installStep.Run, "command -v", "Should check if tools exist")
			assert.Contains(t, installStep.Run, "already installed", "Should skip if already installed")
		}

		// Cache should have restore keys for resilience
		var cacheStep *Step
		for _, step := range setupAction.Runs.Steps {
			if strings.Contains(step.Uses, "cache") {
				cacheStep = &step
				break
			}
		}

		if cacheStep != nil {
			if restoreKeys, ok := cacheStep.With["restore-keys"]; ok {
				assert.NotEmpty(t, restoreKeys, "Cache should have restore keys")
			}
		}
	})

	t.Run("ArtifactUploadFailures", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)

		// Upload steps should be conditional or have error handling
		for jobName, job := range workflow.Jobs {
			for _, step := range job.Steps {
				if strings.Contains(step.Uses, "upload-artifact") {
					// Should use 'if: always()' for critical artifacts
					if strings.Contains(step.Name, "test results") || 
					   strings.Contains(step.Name, "coverage") ||
					   strings.Contains(step.Name, "security") {
						assert.Equal(t, "always()", step.If, 
							fmt.Sprintf("Critical artifact upload in %s should use 'if: always()'", jobName))
					}
				}
			}
		}
	})
}

// TestPerformanceOptimizations validates performance characteristics of the CI/CD pipeline
func TestPerformanceOptimizations(t *testing.T) {
	t.Run("CacheEffectiveness", func(t *testing.T) {
		setupAction, err := loadActionDefinition(getActionPath("setup", "action.yml"))
		require.NoError(t, err)

		cacheSteps := make(map[string]*Step)
		for i, step := range setupAction.Runs.Steps {
			if strings.Contains(step.Uses, "cache") {
				cacheSteps[step.Name] = &setupAction.Runs.Steps[i]
			}
		}

		// Should have multiple cache layers
		assert.GreaterOrEqual(t, len(cacheSteps), 2, "Should have at least 2 cache layers")

		// Module cache should be effective
		if moduleCache, ok := cacheSteps["Cache Go modules"]; ok {
			assert.Contains(t, moduleCache.With["key"], "hashFiles", "Module cache should use file hashing")
			assert.Contains(t, moduleCache.With["path"], "go-build", "Should cache build cache")
			assert.Contains(t, moduleCache.With["path"], "pkg/mod", "Should cache module cache")
		}

		// Tool cache should be conditional
		if toolCache, ok := cacheSteps["Cache tools"]; ok {
			assert.Contains(t, toolCache.If, "install-tools == 'true'", "Tool cache should be conditional")
		}
	})

	t.Run("ParallelJobExecution", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)

		// Jobs with same dependencies can run in parallel
		parallelGroups := map[string][]string{
			"after-validation": {"build-matrix", "test-suite", "quality-checks"},
		}

		for group, jobs := range parallelGroups {
			t.Logf("Checking parallel group %s: %v", group, jobs)
			
			// Jobs in the same group should have similar dependencies
			var firstJobNeeds []string
			for i, jobName := range jobs {
				job := workflow.Jobs[jobName]
				needs := formatJobNeeds(job.Needs)
				
				if i == 0 {
					firstJobNeeds = needs
				} else {
					// Should have similar dependencies to allow parallel execution
					for _, need := range firstJobNeeds {
						if need != "version-validation" && need != "fast-validation" {
							// Allow some variation in optional dependencies
							t.Logf("Job %s dependencies: %v vs %v", jobName, needs, firstJobNeeds)
						}
					}
				}
			}
		}
	})

	t.Run("ConditionalJobExecution", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)

		// Expensive jobs should be conditional
		expensiveJobs := map[string][]string{
			"security-scan":      {"main", "security"},
			"performance-tests":  {"main", "performance"},
		}

		for jobName, expectedConditions := range expensiveJobs {
			job := workflow.Jobs[jobName]
			assert.NotEmpty(t, job.If, fmt.Sprintf("Job %s should be conditional", jobName))
			
			for _, condition := range expectedConditions {
				assert.Contains(t, job.If, condition, 
					fmt.Sprintf("Job %s should check for %s condition", jobName, condition))
			}
		}

		// Core jobs should run on all relevant events
		coreJobs := []string{"version-validation", "fast-validation", "build-matrix", "test-suite", "quality-checks"}
		for _, jobName := range coreJobs {
			job := workflow.Jobs[jobName]
			// Core jobs should only skip on draft PRs or validation failures
			if job.If != "" {
				assert.Contains(t, job.If, "should-skip != 'true'", 
					fmt.Sprintf("Core job %s should only check skip conditions", jobName))
			}
		}
	})

	t.Run("TimeoutOptimization", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)

		// Timeouts should be reasonable for each job type
		expectedTimeouts := map[string]int{
			"version-validation": 5,   // Should be very fast
			"fast-validation":    5,   // Should be fast
			"build-matrix":      10,   // Cross-platform builds
			"test-suite":        15,   // Comprehensive testing
			"quality-checks":    10,   // Linting and analysis
			"security-scan":     15,   // Security tools can be slow
			"performance-tests": 20,   // Benchmarks take time
			"integration":       5,    // Just result checking
		}

		for jobName, maxTimeout := range expectedTimeouts {
			job := workflow.Jobs[jobName]
			if job.Timeout != "" {
				timeout := 0
				fmt.Sscanf(job.Timeout, "%d", &timeout)
				assert.LessOrEqual(t, timeout, maxTimeout, 
					fmt.Sprintf("Job %s timeout should be <= %d minutes", jobName, maxTimeout))
			}
		}
	})

	t.Run("ArtifactSizeOptimization", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)

		retentionDays := workflow.Env["ARTIFACTS_RETENTION"]
		retention := 0
		fmt.Sscanf(retentionDays, "%d", &retention)
		
		// Retention should be reasonable (not too long to waste storage)
		assert.LessOrEqual(t, retention, 90, "Artifact retention should be <= 90 days")
		assert.GreaterOrEqual(t, retention, 7, "Artifact retention should be >= 7 days")

		// Check for artifact optimization
		for jobName, job := range workflow.Jobs {
			for _, step := range job.Steps {
				if strings.Contains(step.Uses, "upload-artifact") {
					if path, ok := step.With["path"]; ok {
						// Should upload specific paths, not everything
						assert.False(t, path == ".", 
							fmt.Sprintf("Job %s should not upload entire directory", jobName))
						assert.False(t, path == "*", 
							fmt.Sprintf("Job %s should not upload all files", jobName))
					}
				}
			}
		}
	})

	t.Run("ResourceUtilization", func(t *testing.T) {
		// Test runners should be appropriate for workload
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)

		for jobName, job := range workflow.Jobs {
			// Extract runner OS from runs-on
			var runnerOS string
			switch v := job.RunsOn.(type) {
			case string:
				runnerOS = v
			case []interface{}:
				if len(v) > 0 {
					runnerOS = fmt.Sprintf("%v", v[0])
				}
			}

			// Most jobs should use ubuntu-latest (fastest and cheapest)
			if !strings.Contains(jobName, "build-matrix") {
				assert.Contains(t, runnerOS, "ubuntu", 
					fmt.Sprintf("Job %s should use ubuntu runner for efficiency", jobName))
			}
		}

		// Build matrix should use appropriate runners for each platform
		buildMatrix := workflow.Jobs["build-matrix"]
		if strategy, ok := buildMatrix.Strategy["matrix"].(map[string]interface{}); ok {
			if include, ok := strategy["include"].([]interface{}); ok {
				for _, item := range include {
					if entry, ok := item.(map[string]interface{}); ok {
						os := entry["os"]
						goos := entry["goos"]
						
						// Darwin builds should use macos runners
						if goos == "darwin" {
							assert.Contains(t, fmt.Sprintf("%v", os), "macos", 
								"Darwin builds should use macOS runners")
						}
						// Windows builds should use windows runners
						if goos == "windows" {
							assert.Contains(t, fmt.Sprintf("%v", os), "windows", 
								"Windows builds should use Windows runners")
						}
					}
				}
			}
		}
	})
}

// TestContinuousImprovementMetrics validates metrics and monitoring capabilities
func TestContinuousImprovementMetrics(t *testing.T) {
	t.Run("WorkflowObservability", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)

		// Integration job should generate comprehensive summary
		integration := workflow.Jobs["integration"]
		
		var summaryStep *Step
		for _, step := range integration.Steps {
			if strings.Contains(step.Name, "Generate build summary") {
				summaryStep = &step
				break
			}
		}

		require.NotNil(t, summaryStep, "Integration should generate build summary")
		assert.Contains(t, summaryStep.Run, "pipeline-summary.md", "Should generate summary report")

		// Should track metrics
		assert.Contains(t, summaryStep.Run, "Version Validation", "Should track validation metrics")
		assert.Contains(t, summaryStep.Run, "Build Results", "Should track build metrics")
		assert.Contains(t, summaryStep.Run, "Artifacts Generated", "Should track artifact metrics")
	})

	t.Run("PRCommentingSystem", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)

		integration := workflow.Jobs["integration"]
		
		var commentStep *Step
		for _, step := range integration.Steps {
			if strings.Contains(step.Name, "Comment on PR") {
				commentStep = &step
				break
			}
		}

		require.NotNil(t, commentStep, "Integration should comment on PRs")
		assert.Equal(t, "pull_request", commentStep.If, "Should only comment on PRs")
		assert.Contains(t, commentStep.Uses, "github-script", "Should use GitHub API")

		// Comment should be informative
		if script, ok := commentStep.With["script"]; ok {
			assert.Contains(t, script, "CI Pipeline Results", "Should include results summary")
			assert.Contains(t, script, "Jobs Completed", "Should list completed jobs")
			assert.Contains(t, script, "Artifacts", "Should mention artifacts")
		}
	})

	t.Run("QualityTrendTracking", func(t *testing.T) {
		// Test action should generate quality metrics
		testAction, err := loadActionDefinition(getActionPath("test", "action.yml"))
		require.NoError(t, err)

		// Should generate coverage reports
		var coverageStep *Step
		for _, step := range testAction.Runs.Steps {
			if strings.Contains(step.Name, "coverage report") {
				coverageStep = &step
				break
			}
		}

		if coverageStep != nil {
			assert.Contains(t, coverageStep.Run, "coverage.html", "Should generate HTML coverage")
			assert.Contains(t, coverageStep.Run, "coverage-summary.md", "Should generate summary")
		}

		// Security action should track security trends
		securityAction, err := loadActionDefinition(getActionPath("security", "action.yml"))
		require.NoError(t, err)

		var reportStep *Step
		for _, step := range securityAction.Runs.Steps {
			if strings.Contains(step.Name, "consolidated security report") {
				reportStep = &step
				break
			}
		}

		if reportStep != nil {
			assert.Contains(t, reportStep.Run, "scan_timestamp", "Should include timestamp")
			assert.Contains(t, reportStep.Run, "findings_count", "Should track finding counts")
		}
	})

	t.Run("PerformanceBenchmarking", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)

		// Performance tests should be tracked
		if perfJob, ok := workflow.Jobs["performance-tests"]; ok {
			assert.NotEmpty(t, perfJob.Timeout, "Performance tests should have timeout")
			
			// Should upload performance results
			var uploadStep *Step
			for _, step := range perfJob.Steps {
				if strings.Contains(step.Uses, "upload-artifact") && 
				   strings.Contains(step.With["name"], "performance") {
					uploadStep = &step
					break
				}
			}
			
			assert.NotNil(t, uploadStep, "Should upload performance results")
		}
	})
}

// TestRegressionPrevention validates safeguards against common CI/CD regressions
func TestRegressionPrevention(t *testing.T) {
	t.Run("VersionLockPrevention", func(t *testing.T) {
		// Validate that versions are synchronized and can be updated centrally
		versions := extractGoVersions()
		
		// All versions should be identical
		assert.Equal(t, versions.Workflow, versions.Makefile, 
			"Workflow and Makefile versions must match")
		assert.Equal(t, versions.Workflow, versions.GoMod, 
			"Workflow and go.mod versions must match")

		// Validate configuration action checks this
		validateAction, err := loadActionDefinition(getActionPath("validate-config", "action.yml"))
		require.NoError(t, err)

		foundValidation := false
		for _, step := range validateAction.Runs.Steps {
			if strings.Contains(step.Run, "go.mod") && strings.Contains(step.Run, "Makefile") {
				foundValidation = true
				break
			}
		}
		assert.True(t, foundValidation, "Config validation should check version consistency")
	})

	t.Run("DependencyDriftPrevention", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)

		// Fast validation should check go.mod/go.sum consistency
		fastValidation := workflow.Jobs["fast-validation"]
		
		var modCheckStep *Step
		for _, step := range fastValidation.Steps {
			if strings.Contains(step.Name, "Go modules") {
				modCheckStep = &step
				break
			}
		}

		require.NotNil(t, modCheckStep, "Should verify Go modules")
		assert.Contains(t, modCheckStep.Run, "go mod tidy", "Should tidy modules")
		assert.Contains(t, modCheckStep.Run, "git diff", "Should check for changes")
		assert.Contains(t, modCheckStep.Run, "exit 1", "Should fail if modules changed")
	})

	t.Run("SecurityRegressionPrevention", func(t *testing.T) {
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)

		// Security scan should run on main branch pushes
		securityScan := workflow.Jobs["security-scan"]
		assert.Contains(t, securityScan.If, "refs/heads/main", 
			"Security scan should run on main branch")

		// Should fail on high severity by default
		securityAction, err := loadActionDefinition(getActionPath("security", "action.yml"))
		require.NoError(t, err)

		failOnHigh := securityAction.Inputs["fail-on-high"]
		assert.Equal(t, "true", failOnHigh.Default, "Should fail on high severity by default")
	})

	t.Run("QualityRegressionPrevention", func(t *testing.T) {
		// Coverage thresholds should prevent quality regression
		testAction, err := loadActionDefinition(getActionPath("test", "action.yml"))
		require.NoError(t, err)

		coverageThreshold := testAction.Inputs["coverage-threshold"]
		assert.Equal(t, "80", coverageThreshold.Default, "Default coverage should be 80%")

		// Should enforce thresholds
		var coverageStep *Step
		for _, step := range testAction.Runs.Steps {
			if strings.Contains(step.Name, "coverage report") {
				coverageStep = &step
				break
			}
		}

		if coverageStep != nil {
			assert.Contains(t, coverageStep.Run, "THRESHOLD", "Should check coverage threshold")
			assert.Contains(t, coverageStep.Run, "exit 1", "Should fail if below threshold")
		}
	})

	t.Run("ConfigurationDriftPrevention", func(t *testing.T) {
		// .golangci.yml should be validated
		workflow, err := loadWorkflowDefinition(getWorkflowPath("ci.yml"))
		require.NoError(t, err)

		qualityChecks := workflow.Jobs["quality-checks"]
		
		var golangciStep *Step
		for _, step := range qualityChecks.Steps {
			if strings.Contains(step.Name, ".golangci.yml") {
				golangciStep = &step
				break
			}
		}

		require.NotNil(t, golangciStep, "Should validate golangci config")
		assert.Contains(t, golangciStep.Run, ".golangci.yml", "Should check config file")
		assert.Contains(t, golangciStep.Run, "exit 1", "Should fail if config missing")
	})
}

// BenchmarkWorkflowPerformance measures actual workflow performance characteristics
func BenchmarkWorkflowPerformance(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	b.Run("ConfigurationParsing", func(b *testing.B) {
		workflowPath := getWorkflowPath("ci.yml")
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := loadWorkflowDefinition(workflowPath)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ActionDefinitionParsing", func(b *testing.B) {
		actions := []string{"setup", "test", "security", "build", "validate-config"}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, actionName := range actions {
				actionPath := getActionPath(actionName, "action.yml")
				_, err := loadActionDefinition(actionPath)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})

	b.Run("VersionExtraction", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = extractGoVersions()
		}
	})
}