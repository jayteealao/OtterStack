// Package cmd provides CLI commands for OtterStack.
package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/jayteealao/otterstack/internal/lock"
	"github.com/jayteealao/otterstack/internal/state"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Version is the current version of OtterStack.
// Can be overridden at build time: go build -ldflags "-X github.com/jayteealao/otterstack/cmd.Version=v1.0.0"
var Version = "v0.2.2"

var (
	cfgFile string
	dataDir string
	verbose bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "otterstack",
	Short: "Git-driven Docker Compose deployment for single VPS",
	Long: `OtterStack is a multi-project, Git-driven Docker Compose deployment tool
designed for single Linux VPS environments.

It manages deployments using git worktrees for atomic releases and provides
rollback capabilities with zero-downtime deployments.`,
	Version: Version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		fmt.Fprintf(os.Stderr, "\nReceived signal %v, shutting down...\n", sig)
		cancel()
	}()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.otterstack/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", "", "data directory (default is $HOME/.otterstack)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "enable verbose output")

	// Bind flags to viper
	viper.BindPFlag("data-dir", rootCmd.PersistentFlags().Lookup("data-dir"))
	viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		configDir := filepath.Join(home, ".otterstack")
		viper.AddConfigPath(configDir)
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	// Read environment variables
	viper.SetEnvPrefix("OTTERSTACK")
	viper.AutomaticEnv()

	// Read config file
	if err := viper.ReadInConfig(); err == nil && verbose {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

// getDataDir returns the data directory, defaulting to $HOME/.otterstack
func getDataDir() (string, error) {
	if dataDir != "" {
		return dataDir, nil
	}
	if d := viper.GetString("data-dir"); d != "" {
		return d, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".otterstack"), nil
}

// initStore initializes and returns the state store.
func initStore() (*state.Store, error) {
	dir, err := getDataDir()
	if err != nil {
		return nil, err
	}

	store, err := state.New(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize store: %w", err)
	}

	return store, nil
}

// initLockManager initializes and returns the lock manager.
func initLockManager() (*lock.Manager, error) {
	dir, err := getDataDir()
	if err != nil {
		return nil, err
	}

	manager, err := lock.NewManager(dir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize lock manager: %w", err)
	}

	return manager, nil
}

// isVerbose returns true if verbose output is enabled.
func isVerbose() bool {
	return verbose || viper.GetBool("verbose")
}

// printVerbose prints a message if verbose mode is enabled.
func printVerbose(format string, args ...interface{}) {
	if isVerbose() {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

// checkContext returns an error if the context is cancelled.
func checkContext(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
