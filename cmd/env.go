package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"

	apperrors "github.com/jayteealao/otterstack/internal/errors"
	"github.com/jayteealao/otterstack/internal/validate"
	"github.com/spf13/cobra"
)

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage project environment variables",
	Long:  `Set, get, list, and unset environment variables for projects.`,
}

var envSetCmd = &cobra.Command{
	Use:   "set <project> KEY=VALUE [KEY=VALUE...]",
	Short: "Set environment variables",
	Long: `Set one or more environment variables for a project.

Variables are stored securely and passed to Docker Compose via --env-file
during deployment.

Examples:
  otterstack env set myapp DATABASE_URL=postgres://localhost/db
  otterstack env set myapp API_KEY=secret123 DEBUG=false`,
	Args: cobra.MinimumNArgs(2),
	RunE: runEnvSet,
}

var envGetCmd = &cobra.Command{
	Use:   "get <project> [KEY]",
	Short: "Get environment variable value",
	Long: `Get the value of an environment variable for a project.

If no KEY is specified, lists all environment variables with values.

Examples:
  otterstack env get myapp DATABASE_URL
  otterstack env get myapp`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runEnvGet,
}

var envListCmd = &cobra.Command{
	Use:     "list <project>",
	Aliases: []string{"ls"},
	Short:   "List environment variables",
	Long: `List all environment variables for a project.

By default, values are masked. Use --show-values to reveal them.

Examples:
  otterstack env list myapp
  otterstack env list myapp --show-values`,
	Args: cobra.ExactArgs(1),
	RunE: runEnvList,
}

var envUnsetCmd = &cobra.Command{
	Use:     "unset <project> KEY [KEY...]",
	Aliases: []string{"rm", "delete"},
	Short:   "Remove environment variables",
	Long: `Remove one or more environment variables from a project.

Examples:
  otterstack env unset myapp DEBUG
  otterstack env unset myapp API_KEY SECRET_KEY`,
	Args: cobra.MinimumNArgs(2),
	RunE: runEnvUnset,
}

var envLoadCmd = &cobra.Command{
	Use:     "load <project> <file>",
	Aliases: []string{"import"},
	Short:   "Load environment variables from a file",
	Long: `Load environment variables from a dotenv file.

The file should contain one KEY=VALUE pair per line.
Lines starting with # are treated as comments and ignored.
Empty lines are also ignored.

Examples:
  otterstack env load myapp .env
  otterstack env load myapp production.env`,
	Args: cobra.ExactArgs(2),
	RunE: runEnvLoad,
}

var showValuesFlag bool

func init() {
	rootCmd.AddCommand(envCmd)
	envCmd.AddCommand(envSetCmd)
	envCmd.AddCommand(envGetCmd)
	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envUnsetCmd)
	envCmd.AddCommand(envLoadCmd)

	envListCmd.Flags().BoolVar(&showValuesFlag, "show-values", false, "show actual values instead of masking")
}

func runEnvSet(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	projectName := args[0]

	store, err := initStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// Get project
	project, err := store.GetProject(ctx, projectName)
	if err != nil {
		if errors.Is(err, apperrors.ErrProjectNotFound) {
			return fmt.Errorf("project %q not found", projectName)
		}
		return err
	}

	// Parse KEY=VALUE pairs
	vars := make(map[string]string)
	for _, arg := range args[1:] {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid format %q: expected KEY=VALUE", arg)
		}

		key := parts[0]
		value := parts[1]

		// Validate key
		if err := validate.EnvKey(key); err != nil {
			return fmt.Errorf("invalid key %q: %w", key, err)
		}

		vars[key] = value
	}

	// Set env vars
	if err := store.SetEnvVars(ctx, project.ID, vars); err != nil {
		return fmt.Errorf("failed to set env vars: %w", err)
	}

	// Print confirmation
	for k := range vars {
		fmt.Printf("Set %s\n", k)
	}

	fmt.Println("\nNote: Redeploy the project for changes to take effect.")
	return nil
}

func runEnvGet(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	projectName := args[0]

	store, err := initStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// Get project
	project, err := store.GetProject(ctx, projectName)
	if err != nil {
		if errors.Is(err, apperrors.ErrProjectNotFound) {
			return fmt.Errorf("project %q not found", projectName)
		}
		return err
	}

	// Get env vars
	vars, err := store.GetEnvVars(ctx, project.ID)
	if err != nil {
		return fmt.Errorf("failed to get env vars: %w", err)
	}

	// If specific key requested
	if len(args) > 1 {
		key := args[1]
		value, ok := vars[key]
		if !ok {
			return fmt.Errorf("environment variable %q not found", key)
		}
		fmt.Println(value)
		return nil
	}

	// Print all vars with values
	if len(vars) == 0 {
		fmt.Println("No environment variables set.")
		return nil
	}

	keys := sortedKeys(vars)
	for _, k := range keys {
		fmt.Printf("%s=%s\n", k, vars[k])
	}

	return nil
}

func runEnvList(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	projectName := args[0]

	store, err := initStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// Get project
	project, err := store.GetProject(ctx, projectName)
	if err != nil {
		if errors.Is(err, apperrors.ErrProjectNotFound) {
			return fmt.Errorf("project %q not found", projectName)
		}
		return err
	}

	// Get env vars
	vars, err := store.GetEnvVars(ctx, project.ID)
	if err != nil {
		return fmt.Errorf("failed to get env vars: %w", err)
	}

	if len(vars) == 0 {
		fmt.Println("No environment variables set.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tVALUE")
	fmt.Fprintln(w, "---\t-----")

	keys := sortedKeys(vars)
	for _, k := range keys {
		value := vars[k]
		if !showValuesFlag {
			value = maskValue(value)
		}
		fmt.Fprintf(w, "%s\t%s\n", k, value)
	}
	w.Flush()

	return nil
}

func runEnvUnset(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	projectName := args[0]

	store, err := initStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// Get project
	project, err := store.GetProject(ctx, projectName)
	if err != nil {
		if errors.Is(err, apperrors.ErrProjectNotFound) {
			return fmt.Errorf("project %q not found", projectName)
		}
		return err
	}

	// Delete each key
	for _, key := range args[1:] {
		if err := store.DeleteEnvVar(ctx, project.ID, key); err != nil {
			return fmt.Errorf("failed to unset %s: %w", key, err)
		}
		fmt.Printf("Unset %s\n", key)
	}

	fmt.Println("\nNote: Redeploy the project for changes to take effect.")
	return nil
}

func runEnvLoad(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	projectName := args[0]
	filePath := args[1]

	store, err := initStore()
	if err != nil {
		return err
	}
	defer store.Close()

	// Get project
	project, err := store.GetProject(ctx, projectName)
	if err != nil {
		if errors.Is(err, apperrors.ErrProjectNotFound) {
			return fmt.Errorf("project %q not found", projectName)
		}
		return err
	}

	// Open and parse file
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	vars := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid format at line %d: expected KEY=VALUE", lineNum)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove surrounding quotes from value if present
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		// Validate key
		if err := validate.EnvKey(key); err != nil {
			return fmt.Errorf("invalid key %q at line %d: %w", key, lineNum, err)
		}

		vars[key] = value
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if len(vars) == 0 {
		fmt.Println("No environment variables found in file.")
		return nil
	}

	// Set env vars
	if err := store.SetEnvVars(ctx, project.ID, vars); err != nil {
		return fmt.Errorf("failed to set env vars: %w", err)
	}

	// Print confirmation
	fmt.Printf("Loaded %d environment variable(s) from %s:\n", len(vars), filePath)
	for _, k := range sortedKeys(vars) {
		fmt.Printf("  %s\n", k)
	}

	fmt.Println("\nNote: Redeploy the project for changes to take effect.")
	return nil
}

// maskValue returns a masked version of the value for display.
func maskValue(value string) string {
	if len(value) == 0 {
		return ""
	}
	if len(value) <= 4 {
		return "****"
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

// sortedKeys returns the keys of a map in sorted order.
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
