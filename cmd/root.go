package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/claude-code/env-switcher/internal/config"
	"github.com/claude-code/env-switcher/internal/launcher"
	"github.com/claude-code/env-switcher/internal/parser"
	"github.com/claude-code/env-switcher/internal/ui"
	"github.com/claude-code/env-switcher/pkg/types"
)

var (
	cfgFile    string
	envName    string
	verbose    bool
	noInteractive bool
	version    = "1.1.0" // Updated version with pass-through and model support
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "cce [flags] [claude-code-args...]",
	Short: "Claude Code Environment Switcher",
	Long: `Claude Code Environment Switcher (CCE) is a tool for managing multiple 
Claude Code API endpoint configurations and seamlessly switching between them.

When run without arguments, CCE will:
1. Display an interactive environment selection menu (if multiple environments exist)
2. Launch Claude Code with the selected environment configuration
3. Launch Claude Code directly if no environments are configured

All arguments after the flags will be passed through to Claude Code.`,
	RunE: runRoot,
	Args: cobra.ArbitraryArgs,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.claude-code-env/config.json)")
	rootCmd.PersistentFlags().StringVarP(&envName, "env", "e", "", "environment name to use")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().BoolVar(&noInteractive, "no-interactive", false, "disable interactive mode")
	
	// Add version command
	rootCmd.AddCommand(&cobra.Command{
		Use:   "version",
		Short: "Print the version number",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Claude Code Environment Switcher v%s\n", version)
		},
	})
}

// runRoot executes the main CCE logic with pass-through support
func runRoot(cmd *cobra.Command, args []string) error {
	// Initialize components
	configManager, err := config.NewFileConfigManager()
	if err != nil {
		return err
	}

	ui := ui.NewTerminalUI()
	launcher := launcher.NewSystemLauncher()
	
	// Initialize argument analysis components
	registry := parser.NewFlagRegistry()
	analyzer := parser.NewArgumentAnalyzer(registry)
	delegationEngine := parser.NewDelegationEngine(analyzer, registry)

	// Analyze arguments to determine routing strategy
	analysis, err := analyzer.AnalyzeArguments(args)
	if err != nil {
		return fmt.Errorf("failed to analyze arguments: %w", err)
	}

	if verbose {
		ui.ShowInfo(fmt.Sprintf("Argument analysis: CCE flags=%t, Claude flags=%t, requires passthrough=%t", 
			analysis.HasCCEFlags, analysis.HasClaudeFlags, analysis.RequiresPassthrough))
	}

	// Handle special cases first
	if analysis.IsHelpRequested {
		return showCombinedHelp(ui, launcher)
	}

	if analysis.IsVersionRequested {
		fmt.Printf("Claude Code Environment Switcher v%s\n", version)
		return nil
	}

	// Load configuration
	cfg, err := configManager.Load()
	if err != nil {
		return err
	}

	// Determine if we should delegate to Claude CLI
	if delegationEngine.ShouldDelegate(analysis) {
		return handleDelegation(delegationEngine, launcher, cfg, args, ui)
	}

	// Handle CCE-specific commands (existing behavior)
	return handleCCECommand(configManager, ui, launcher, cfg, args)
}

// handleDelegation manages delegation to Claude CLI
func handleDelegation(engine *parser.DelegationEngine, launcher *launcher.SystemLauncher, cfg *types.Config, args []string, ui *ui.TerminalUI) error {
	// Determine environment to use
	var selectedEnv *types.Environment
	
	// Extract CCE flags to see if environment is specified
	analyzer := engine.Analyzer
	cceFlags, _, err := analyzer.ExtractCCEFlags(args)
	if err != nil {
		return fmt.Errorf("failed to extract CCE flags: %w", err)
	}

	// Use environment selection logic
	if cceFlags.Environment != "" {
		// Environment specified via flag
		env, exists := cfg.Environments[cceFlags.Environment]
		if !exists {
			return fmt.Errorf("environment '%s' not found", cceFlags.Environment)
		}
		selectedEnv = &env
	} else if len(cfg.Environments) == 1 {
		// Only one environment, use it automatically
		for _, env := range cfg.Environments {
			selectedEnv = &env
			break
		}
	} else if len(cfg.Environments) > 1 && !cceFlags.NoInteractive {
		// Multiple environments, show selection menu
		var err error
		selectedEnv, err = selectEnvironment(ui, cfg)
		if err != nil {
			return err
		}
	} else if cfg.DefaultEnv != "" {
		// Use default environment
		if env, exists := cfg.Environments[cfg.DefaultEnv]; exists {
			selectedEnv = &env
		}
	}

	// Prepare delegation plan
	plan, err := engine.PrepareDelegation(selectedEnv, args)
	if err != nil {
		return fmt.Errorf("failed to prepare delegation: %w", err)
	}

	// Enable pass-through mode and delegate
	launcher.SetPassthroughMode(true)
	
	if cceFlags.Verbose {
		ui.ShowInfo(fmt.Sprintf("Delegating to Claude CLI with environment: %s", 
			getEnvironmentDisplayName(selectedEnv)))
		
		if selectedEnv != nil && selectedEnv.Model != "" {
			ui.ShowInfo(fmt.Sprintf("Using model: %s", selectedEnv.Model))
		}
	}

	return launcher.LaunchWithDelegation(plan)
}

// handleCCECommand handles CCE-specific commands (preserves existing behavior)
func handleCCECommand(configManager types.ConfigManager, ui *ui.TerminalUI, launcher types.ClaudeCodeLauncher, cfg *types.Config, args []string) error {
	// Determine which environment to use
	var selectedEnv *types.Environment

	if len(cfg.Environments) == 0 {
		// No environments configured, launch Claude Code directly
		if verbose {
			ui.ShowInfo("No environments configured, launching Claude Code directly")
		}
		return launcher.Launch(nil, args)
	}

	if envName != "" {
		// Environment specified via flag
		env, exists := cfg.Environments[envName]
		if !exists {
			return fmt.Errorf("environment '%s' not found", envName)
		}
		selectedEnv = &env
	} else if len(cfg.Environments) == 1 {
		// Only one environment, use it automatically
		for _, env := range cfg.Environments {
			selectedEnv = &env
			break
		}
		if verbose {
			ui.ShowInfo(fmt.Sprintf("Using environment: %s", selectedEnv.Name))
		}
	} else if !noInteractive {
		// Multiple environments, show selection menu
		var err error
		selectedEnv, err = selectEnvironment(ui, cfg)
		if err != nil {
			return err
		}
	} else {
		// Non-interactive mode with multiple environments and no env specified
		if cfg.DefaultEnv != "" {
			env, exists := cfg.Environments[cfg.DefaultEnv]
			if exists {
				selectedEnv = &env
			}
		}
		
		if selectedEnv == nil {
			return fmt.Errorf("multiple environments available but no environment specified. Use --env flag or run interactively")
		}
	}

	// Update default environment if selection was made interactively
	if selectedEnv != nil && selectedEnv.Name != cfg.DefaultEnv && !noInteractive && envName == "" {
		cfg.DefaultEnv = selectedEnv.Name
		if err := configManager.Save(cfg); err != nil {
			// Don't fail the launch if we can't save the default, just warn
			if verbose {
				ui.ShowWarning(fmt.Sprintf("Warning: Could not save default environment: %v", err))
			}
		}
	}

	// Launch Claude Code with selected environment
	if verbose && selectedEnv != nil {
		ui.ShowInfo(fmt.Sprintf("Launching Claude Code with environment: %s", selectedEnv.Name))
		if selectedEnv.Model != "" {
			ui.ShowInfo(fmt.Sprintf("Using model: %s", selectedEnv.Model))
		}
	}

	return launcher.Launch(selectedEnv, args)
}

// showCombinedHelp displays help information for both CCE and Claude CLI
func showCombinedHelp(ui *ui.TerminalUI, launcher types.ClaudeCodeLauncher) error {
	// Show CCE help first
	fmt.Println("Claude Code Environment Switcher (CCE) - Enhanced CLI wrapper")
	fmt.Println()
	fmt.Println("CCE acts as a drop-in replacement for Claude CLI, adding environment management")
	fmt.Println("and model configuration capabilities while preserving all Claude CLI functionality.")
	fmt.Println()
	fmt.Println("CCE-specific flags:")
	fmt.Println("  --env, -e string      Environment name to use")
	fmt.Println("  --config string       Config file path") 
	fmt.Println("  --verbose, -v         Verbose output")
	fmt.Println("  --no-interactive      Disable interactive mode")
	fmt.Println("  --help, -h            Show this help")
	fmt.Println("  --version             Show version")
	fmt.Println()
	fmt.Println("All other flags and arguments are passed through to Claude CLI.")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  cce                              # Interactive environment selection")
	fmt.Println("  cce --env production             # Use specific environment")
	fmt.Println("  cce -r \"You are a helpful assistant\" # Pass-through to Claude CLI")
	fmt.Println("  cce --env staging -r \"Debug this code\" # Environment + Claude flags")
	fmt.Println()
	
	// Try to show Claude CLI help as well
	if err := launcher.ValidateClaudeCode(); err == nil {
		fmt.Println("Claude CLI help:")
		fmt.Println("================")
		
		// Execute claude --help to show Claude CLI options
		// Note: This is a simplified approach - in production, you might want
		// to capture and format the output more carefully
		launcher.Launch(nil, []string{"--help"})
	} else {
		fmt.Println("Claude CLI not found - install Claude CLI to see additional options")
	}

	return nil
}

// getEnvironmentDisplayName returns a display-friendly environment name
func getEnvironmentDisplayName(env *types.Environment) string {
	if env == nil {
		return "none"
	}
	if env.Name != "" {
		return env.Name
	}
	return env.BaseURL
}

// selectEnvironment shows an interactive environment selection menu
func selectEnvironment(ui *ui.TerminalUI, cfg *types.Config) (*types.Environment, error) {
	var items []types.SelectItem

	// Build selection items with model information
	for name, env := range cfg.Environments {
		description := env.Description
		if description == "" {
			description = env.BaseURL
		}
		
		// Add model information to description if available
		if env.Model != "" {
			description = fmt.Sprintf("%s (Model: %s)", description, env.Model)
		} else {
			description = fmt.Sprintf("%s (Default model)", description)
		}
		
		items = append(items, types.SelectItem{
			Label:       name,
			Description: description,
			Value:       env,
		})
	}

	// Show selection menu
	index, _, err := ui.Select("Select Claude Code environment", items)
	if err != nil {
		return nil, err
	}

	// Return selected environment
	selectedEnv := items[index].Value.(types.Environment)
	return &selectedEnv, nil
}