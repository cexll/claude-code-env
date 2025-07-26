package parser

import (
	"fmt"
	"strings"
)

// FlagRegistry maintains classification of CCE and Claude CLI flags
type FlagRegistry struct {
	cceFlags      map[string]FlagInfo
	claudeFlags   map[string]FlagInfo
	conflictFlags map[string]ConflictInfo
	flagAliases   map[string]string // Maps short flags to long flags
}

// FlagInfo contains information about a specific flag
type FlagInfo struct {
	Name        string
	TakesValue  bool
	Required    bool
	Description string
	Category    FlagCategory
}

// ConflictInfo describes how to handle flag conflicts
type ConflictInfo struct {
	CCEFlag    string
	ClaudeFlag string
	Resolution ConflictResolution
	Message    string
}

// FlagCategory categorizes flags by their function
type FlagCategory int

const (
	ConfigurationFlag FlagCategory = iota
	BehaviorFlag
	OutputFlag
	AuthenticationFlag
	NetworkFlag
	HelpFlag
)

// NewFlagRegistry creates and initializes a new flag registry
func NewFlagRegistry() *FlagRegistry {
	registry := &FlagRegistry{
		cceFlags:      make(map[string]FlagInfo),
		claudeFlags:   make(map[string]FlagInfo),
		conflictFlags: make(map[string]ConflictInfo),
		flagAliases:   make(map[string]string),
	}

	registry.initializeCCEFlags()
	registry.initializeClaudeFlags()
	registry.initializeConflicts()
	registry.initializeAliases()

	return registry
}

// initializeCCEFlags registers all known CCE-specific flags
func (r *FlagRegistry) initializeCCEFlags() {
	cceFlags := []FlagInfo{
		{
			Name:        "--env",
			TakesValue:  true,
			Required:    false,
			Description: "Environment name to use",
			Category:    ConfigurationFlag,
		},
		{
			Name:        "--config",
			TakesValue:  true,
			Required:    false,
			Description: "Config file path",
			Category:    ConfigurationFlag,
		},
		{
			Name:        "--verbose",
			TakesValue:  false,
			Required:    false,
			Description: "Verbose output",
			Category:    OutputFlag,
		},
		{
			Name:        "--no-interactive",
			TakesValue:  false,
			Required:    false,
			Description: "Disable interactive mode",
			Category:    BehaviorFlag,
		},
		{
			Name:        "--help",
			TakesValue:  false,
			Required:    false,
			Description: "Show help information",
			Category:    HelpFlag,
		},
		{
			Name:        "--version",
			TakesValue:  false,
			Required:    false,
			Description: "Show version information",
			Category:    HelpFlag,
		},
	}

	for _, flag := range cceFlags {
		r.cceFlags[flag.Name] = flag
	}
}

// initializeClaudeFlags registers known Claude CLI flags
func (r *FlagRegistry) initializeClaudeFlags() {
	// Common Claude CLI flags based on typical patterns
	claudeFlags := []FlagInfo{
		{
			Name:        "-r",
			TakesValue:  true,
			Required:    false,
			Description: "Role/instruction for Claude",
			Category:    BehaviorFlag,
		},
		{
			Name:        "--role",
			TakesValue:  true,
			Required:    false,
			Description: "Role/instruction for Claude",
			Category:    BehaviorFlag,
		},
		{
			Name:        "--model",
			TakesValue:  true,
			Required:    false,
			Description: "Model to use for Claude",
			Category:    ConfigurationFlag,
		},
		{
			Name:        "--temperature",
			TakesValue:  true,
			Required:    false,
			Description: "Temperature setting for Claude",
			Category:    ConfigurationFlag,
		},
		{
			Name:        "--max-tokens",
			TakesValue:  true,
			Required:    false,
			Description: "Maximum tokens for Claude response",
			Category:    ConfigurationFlag,
		},
		{
			Name:        "--output",
			TakesValue:  true,
			Required:    false,
			Description: "Output file path",
			Category:    OutputFlag,
		},
		{
			Name:        "--json",
			TakesValue:  false,
			Required:    false,
			Description: "Output in JSON format",
			Category:    OutputFlag,
		},
		{
			Name:        "--stream",
			TakesValue:  false,
			Required:    false,
			Description: "Stream response",
			Category:    BehaviorFlag,
		},
		{
			Name:        "--no-stream",
			TakesValue:  false,
			Required:    false,
			Description: "Disable streaming",
			Category:    BehaviorFlag,
		},
		{
			Name:        "--system",
			TakesValue:  true,
			Required:    false,
			Description: "System message for Claude",
			Category:    BehaviorFlag,
		},
		{
			Name:        "--context",
			TakesValue:  true,
			Required:    false,
			Description: "Context for Claude conversation",
			Category:    BehaviorFlag,
		},
		{
			Name:        "--input",
			TakesValue:  true,
			Required:    false,
			Description: "Input file path",
			Category:    ConfigurationFlag,
		},
		{
			Name:        "--timeout",
			TakesValue:  true,
			Required:    false,
			Description: "Request timeout",
			Category:    NetworkFlag,
		},
		{
			Name:        "--debug",
			TakesValue:  false,
			Required:    false,
			Description: "Debug mode",
			Category:    OutputFlag,
		},
		{
			Name:        "--quiet",
			TakesValue:  false,
			Required:    false,
			Description: "Quiet mode",
			Category:    OutputFlag,
		},
	}

	for _, flag := range claudeFlags {
		r.claudeFlags[flag.Name] = flag
	}
}

// initializeConflicts defines known flag conflicts and their resolutions
func (r *FlagRegistry) initializeConflicts() {
	conflicts := []ConflictInfo{
		{
			CCEFlag:    "--verbose",
			ClaudeFlag: "--verbose",
			Resolution: CCETakesPrecedence,
			Message:    "CCE verbose flag takes precedence over Claude CLI verbose flag",
		},
		{
			CCEFlag:    "--help",
			ClaudeFlag: "--help",
			Resolution: CCETakesPrecedence,
			Message:    "CCE will show combined help including Claude CLI options",
		},
		{
			CCEFlag:    "--version",
			ClaudeFlag: "--version",
			Resolution: CCETakesPrecedence,
			Message:    "CCE version flag takes precedence",
		},
	}

	for _, conflict := range conflicts {
		r.conflictFlags[conflict.CCEFlag] = conflict
	}
}

// initializeAliases maps short flags to their long equivalents
func (r *FlagRegistry) initializeAliases() {
	aliases := map[string]string{
		"-e": "--env",
		"-v": "--verbose",
		"-h": "--help",
	}

	for short, long := range aliases {
		r.flagAliases[short] = long
	}
}

// IsCCEFlag checks if a flag is a CCE-specific flag
func (r *FlagRegistry) IsCCEFlag(flag string) bool {
	// Normalize flag name
	normalized := r.normalizeFlag(flag)
	_, exists := r.cceFlags[normalized]
	return exists
}

// IsClaudeFlag checks if a flag is a Claude CLI flag
func (r *FlagRegistry) IsClaudeFlag(flag string) bool {
	// Normalize flag name
	normalized := r.normalizeFlag(flag)
	_, exists := r.claudeFlags[normalized]
	return exists
}

// CCEFlagTakesValue checks if a CCE flag takes a value
func (r *FlagRegistry) CCEFlagTakesValue(flag string) bool {
	normalized := r.normalizeFlag(flag)
	if info, exists := r.cceFlags[normalized]; exists {
		return info.TakesValue
	}
	return false
}

// ClaudeFlagTakesValue checks if a Claude flag takes a value
func (r *FlagRegistry) ClaudeFlagTakesValue(flag string) bool {
	normalized := r.normalizeFlag(flag)
	if info, exists := r.claudeFlags[normalized]; exists {
		return info.TakesValue
	}
	return false
}

// GetConflictInfo returns conflict information for a flag
func (r *FlagRegistry) GetConflictInfo(flag string) (ConflictInfo, bool) {
	normalized := r.normalizeFlag(flag)
	conflict, exists := r.conflictFlags[normalized]
	return conflict, exists
}

// normalizeFlag converts flag aliases to their canonical form
func (r *FlagRegistry) normalizeFlag(flag string) string {
	if longForm, exists := r.flagAliases[flag]; exists {
		return longForm
	}
	return flag
}

// GetCCEFlags returns all registered CCE flags
func (r *FlagRegistry) GetCCEFlags() map[string]FlagInfo {
	result := make(map[string]FlagInfo)
	for name, info := range r.cceFlags {
		result[name] = info
	}
	return result
}

// GetClaudeFlags returns all registered Claude CLI flags
func (r *FlagRegistry) GetClaudeFlags() map[string]FlagInfo {
	result := make(map[string]FlagInfo)
	for name, info := range r.claudeFlags {
		result[name] = info
	}
	return result
}

// AddCCEFlag dynamically adds a CCE flag to the registry
func (r *FlagRegistry) AddCCEFlag(flag FlagInfo) {
	r.cceFlags[flag.Name] = flag
}

// AddClaudeFlag dynamically adds a Claude CLI flag to the registry
func (r *FlagRegistry) AddClaudeFlag(flag FlagInfo) {
	r.claudeFlags[flag.Name] = flag
}

// IsKnownFlag checks if a flag is known (either CCE or Claude)
func (r *FlagRegistry) IsKnownFlag(flag string) bool {
	return r.IsCCEFlag(flag) || r.IsClaudeFlag(flag)
}

// GetFlagCategory returns the category of a flag
func (r *FlagRegistry) GetFlagCategory(flag string) (FlagCategory, bool) {
	normalized := r.normalizeFlag(flag)

	if info, exists := r.cceFlags[normalized]; exists {
		return info.Category, true
	}

	if info, exists := r.claudeFlags[normalized]; exists {
		return info.Category, true
	}

	return ConfigurationFlag, false
}

// GetFlagDescription returns the description of a flag
func (r *FlagRegistry) GetFlagDescription(flag string) (string, bool) {
	normalized := r.normalizeFlag(flag)

	if info, exists := r.cceFlags[normalized]; exists {
		return info.Description, true
	}

	if info, exists := r.claudeFlags[normalized]; exists {
		return info.Description, true
	}

	return "", false
}

// ValidateFlag performs basic validation on a flag
func (r *FlagRegistry) ValidateFlag(flag string, value string) error {
	normalized := r.normalizeFlag(flag)

	var info FlagInfo
	var exists bool

	if info, exists = r.cceFlags[normalized]; !exists {
		if info, exists = r.claudeFlags[normalized]; !exists {
			return nil // Unknown flags are not validated here
		}
	}

	// Check if flag requires a value but none was provided
	if info.TakesValue && strings.TrimSpace(value) == "" {
		return fmt.Errorf("flag %s requires a value", flag)
	}

	// Check if flag doesn't take a value but one was provided
	if !info.TakesValue && value != "" {
		return fmt.Errorf("flag %s does not take a value", flag)
	}

	return nil
}

// GetAllAliases returns all flag aliases
func (r *FlagRegistry) GetAllAliases() map[string]string {
	result := make(map[string]string)
	for short, long := range r.flagAliases {
		result[short] = long
	}
	return result
}
