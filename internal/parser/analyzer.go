package parser

import (
	"fmt"
	"regexp"
	"strings"
)

// ArgumentAnalyzer implements argument parsing and classification for command routing
type ArgumentAnalyzer struct {
	registry *FlagRegistry
}

// NewArgumentAnalyzer creates a new ArgumentAnalyzer instance
func NewArgumentAnalyzer(registry *FlagRegistry) *ArgumentAnalyzer {
	return &ArgumentAnalyzer{
		registry: registry,
	}
}

// ArgumentAnalysis contains the results of analyzing command-line arguments
type ArgumentAnalysis struct {
	HasCCEFlags      bool
	HasClaudeFlags   bool
	RequiresPassthrough bool
	EnvironmentHints []string
	IsHelpRequested  bool
	IsVersionRequested bool
}

// FlagClassification categorizes flags into CCE and Claude CLI flags
type FlagClassification struct {
	CCEFlags    []string
	ClaudeFlags []string
	Conflicts   []FlagConflict
	Unknown     []string
}

// FlagConflict represents a conflict between CCE and Claude CLI flags
type FlagConflict struct {
	Flag        string
	CCEValue    string
	ClaudeValue string
	Resolution  ConflictResolution
}

// ConflictResolution defines how flag conflicts are resolved
type ConflictResolution int

const (
	CCETakesPrecedence ConflictResolution = iota
	ClaudeTakesPrecedence
	ConflictError
)

// CCEFlags represents parsed CCE-specific flags
type CCEFlags struct {
	Environment   string
	Config        string
	Verbose       bool
	NoInteractive bool
	ShowVersion   bool
	ShowHelp      bool
}

// AnalyzeArguments performs comprehensive analysis of command-line arguments
func (a *ArgumentAnalyzer) AnalyzeArguments(args []string) (*ArgumentAnalysis, error) {
	if len(args) == 0 {
		return &ArgumentAnalysis{
			HasCCEFlags:         false,
			HasClaudeFlags:      false,
			RequiresPassthrough: false,
			EnvironmentHints:    []string{},
		}, nil
	}

	classification, err := a.ClassifyFlags(args)
	if err != nil {
		return nil, fmt.Errorf("flag classification failed: %w", err)
	}

	analysis := &ArgumentAnalysis{
		HasCCEFlags:        len(classification.CCEFlags) > 0,
		HasClaudeFlags:     len(classification.ClaudeFlags) > 0,
		EnvironmentHints:   a.extractEnvironmentHints(args),
		IsHelpRequested:    a.isHelpRequested(args),
		IsVersionRequested: a.isVersionRequested(args),
	}

	// Determine if pass-through is required
	analysis.RequiresPassthrough = analysis.HasClaudeFlags || 
		(!analysis.HasCCEFlags && !analysis.IsHelpRequested && !analysis.IsVersionRequested && len(args) > 0)

	return analysis, nil
}

// ClassifyFlags categorizes flags into CCE and Claude CLI flags
func (a *ArgumentAnalyzer) ClassifyFlags(args []string) (*FlagClassification, error) {
	classification := &FlagClassification{
		CCEFlags:    []string{},
		ClaudeFlags: []string{},
		Conflicts:   []FlagConflict{},
		Unknown:     []string{},
	}

	i := 0
	for i < len(args) {
		arg := args[i]
		
		// Skip non-flag arguments
		if !strings.HasPrefix(arg, "-") {
			i++
			continue
		}

		// Parse flag and potential value
		var flag, value string
		var hasValue bool

		if strings.Contains(arg, "=") {
			// Format: --flag=value
			parts := strings.SplitN(arg, "=", 2)
			flag = parts[0]
			value = parts[1]
			hasValue = true
		} else {
			flag = arg
			// Check if next argument is a value (not starting with -)
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				value = args[i+1]
				hasValue = true
				i++ // Skip the value argument
			}
		}

		// Classify the flag
		if a.registry.IsCCEFlag(flag) {
			classification.CCEFlags = append(classification.CCEFlags, flag)
			if hasValue && a.registry.CCEFlagTakesValue(flag) {
				// Value is part of CCE flag, don't classify separately
			}
		} else if a.registry.IsClaudeFlag(flag) {
			classification.ClaudeFlags = append(classification.ClaudeFlags, flag)
		} else {
			// Check for conflicts
			if a.registry.IsCCEFlag(flag) && a.registry.IsClaudeFlag(flag) {
				conflict := FlagConflict{
					Flag:       flag,
					Resolution: a.resolveConflict(flag),
				}
				classification.Conflicts = append(classification.Conflicts, conflict)
			} else {
				classification.Unknown = append(classification.Unknown, flag)
			}
		}

		i++
	}

	return classification, nil
}

// ExtractCCEFlags extracts CCE-specific flags and returns remaining arguments
func (a *ArgumentAnalyzer) ExtractCCEFlags(args []string) (*CCEFlags, []string, error) {
	cceFlags := &CCEFlags{}
	var remainingArgs []string

	i := 0
	for i < len(args) {
		arg := args[i]

		if !strings.HasPrefix(arg, "-") {
			// Non-flag argument, add to remaining
			remainingArgs = append(remainingArgs, arg)
			i++
			continue
		}

		var flag, value string
		var hasValue bool

		if strings.Contains(arg, "=") {
			parts := strings.SplitN(arg, "=", 2)
			flag = parts[0]
			value = parts[1]
			hasValue = true
		} else {
			flag = arg
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				value = args[i+1]
				hasValue = true
			}
		}

		// Process CCE flags
		processed := false
		switch flag {
		case "--env", "-e":
			if hasValue {
				cceFlags.Environment = value
				if !strings.Contains(arg, "=") {
					i++ // Skip value argument
				}
				processed = true
			}
		case "--config":
			if hasValue {
				cceFlags.Config = value
				if !strings.Contains(arg, "=") {
					i++ // Skip value argument
				}
				processed = true
			}
		case "--verbose", "-v":
			cceFlags.Verbose = true
			processed = true
		case "--no-interactive":
			cceFlags.NoInteractive = true
			processed = true
		case "--version":
			cceFlags.ShowVersion = true
			processed = true
		case "--help", "-h":
			cceFlags.ShowHelp = true
			processed = true
		}

		if !processed {
			// Not a CCE flag, add to remaining arguments
			remainingArgs = append(remainingArgs, arg)
			if hasValue && !strings.Contains(arg, "=") {
				remainingArgs = append(remainingArgs, value)
				i++ // Skip value argument
			}
		}

		i++
	}

	return cceFlags, remainingArgs, nil
}

// extractEnvironmentHints looks for environment-related hints in arguments
func (a *ArgumentAnalyzer) extractEnvironmentHints(args []string) []string {
	var hints []string
	
	// Look for patterns that might indicate environment preferences
	envPattern := regexp.MustCompile(`(?i)(prod|production|staging|dev|development|test|local)`)
	
	for _, arg := range args {
		if matches := envPattern.FindAllString(arg, -1); len(matches) > 0 {
			hints = append(hints, matches...)
		}
	}
	
	return hints
}

// isHelpRequested checks if help is being requested
func (a *ArgumentAnalyzer) isHelpRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return true
		}
	}
	return false
}

// isVersionRequested checks if version is being requested
func (a *ArgumentAnalyzer) isVersionRequested(args []string) bool {
	for _, arg := range args {
		if arg == "--version" || arg == "version" {
			return true
		}
	}
	return false
}

// resolveConflict determines how to resolve flag conflicts
func (a *ArgumentAnalyzer) resolveConflict(flag string) ConflictResolution {
	// CCE flags generally take precedence to maintain control
	// This can be configured per flag if needed
	return CCETakesPrecedence
}

// PreserveArgumentStructure ensures complex arguments are preserved correctly
func (a *ArgumentAnalyzer) PreserveArgumentStructure(args []string) []string {
	preserved := make([]string, len(args))
	
	for i, arg := range args {
		// Preserve quotes and escape sequences
		if a.needsQuoting(arg) {
			preserved[i] = a.quoteArgument(arg)
		} else {
			preserved[i] = arg
		}
	}
	
	return preserved
}

// needsQuoting determines if an argument needs quoting
func (a *ArgumentAnalyzer) needsQuoting(arg string) bool {
	// Check for spaces, special characters, or existing quotes
	specialChars := regexp.MustCompile(`[\s"'\\$`+"`"+`|&;(){}[\]*?<>~]`)
	return specialChars.MatchString(arg)
}

// quoteArgument properly quotes an argument for shell safety
func (a *ArgumentAnalyzer) quoteArgument(arg string) string {
	// If argument contains single quotes, use double quotes
	if strings.Contains(arg, "'") {
		return `"` + strings.ReplaceAll(arg, `"`, `\"`) + `"`
	}
	// Otherwise use single quotes for simplicity
	return "'" + arg + "'"
}