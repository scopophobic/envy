package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/envo/cli/internal/api"
	"github.com/envo/cli/internal/store"
	"github.com/spf13/cobra"
)

func newRunCmd(deps *rootDeps) *cobra.Command {
	var (
		orgSel     string
		projectSel string
		envSel     string
		dir        string
		keys       []string
		purpose    string
	)

	cmd := &cobra.Command{
		Use:   "run --project <project> --env <env> -- <command> [args...]",
		Short: "Fetch secrets and inject them as env vars into a child process (never writes to disk)",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			agentMode := deps.cfg.AgentToken != ""
			if deps.tokens == nil && !agentMode {
				return fmt.Errorf("not logged in; run `envo login`")
			}

			if dir == "" {
				cwd, _ := os.Getwd()
				dir = cwd
			}
			dir, _ = filepath.Abs(dir)

			ctx, cancel := context.WithTimeout(cmd.Context(), 90*time.Second)
			defer cancel()

			var secrets map[string]string
			if agentMode {
				client := api.NewAgentClient(deps.cfg.APIBaseURL, deps.cfg.AgentToken)
				resolved, err := client.ResolveAgentSecrets(ctx, api.ResolveAgentSecretsRequest{
					Project: projectSel, Environment: envSel, Keys: keys, Purpose: purpose,
					SessionID: fmt.Sprintf("envo-run-%d", os.Getpid()),
				})
				if err != nil {
					return err
				}
				secrets = resolved.Secrets
			} else {
				client := api.NewClient(deps.cfg.APIBaseURL, deps.tokens)
				t, err := client.EnsureAccessToken(ctx)
				if err != nil {
					return err
				}
				_ = store.SaveTokens(*t)

				orgID, err := resolveOrgID(ctx, client, orgSel)
				if err != nil {
					return err
				}
				projectID, err := resolveProjectID(ctx, client, orgID, projectSel)
				if err != nil {
					return err
				}
				envID, err := resolveEnvID(ctx, client, projectID, envSel)
				if err != nil {
					return err
				}

				secrets, err = client.ExportEnvironmentSecrets(ctx, envID)
				if err != nil {
					return err
				}
			}

			// Inject secrets directly into the child process env — never write to disk
			child := exec.CommandContext(cmd.Context(), args[0], args[1:]...)
			child.Dir = dir
			child.Stdout = os.Stdout
			child.Stderr = os.Stderr
			child.Stdin = os.Stdin
			child.Env = os.Environ()
			if agentMode {
				// The broker consumes ENVO_TOKEN; the child receives only the
				// approved values, not a reusable credential that could ask for more.
				child.Env = withoutEnvKey(child.Env, "ENVO_TOKEN")
			}
			for k, v := range secrets {
				child.Env = append(child.Env, k+"="+v)
			}

			fmt.Fprintf(os.Stderr, "envo: injecting %d secrets into %s\n", len(secrets), args[0])
			return child.Run()
		},
	}

	cmd.Flags().StringVar(&orgSel, "org", "", "Organization/workspace id or name (default: personal vault)")
	cmd.Flags().StringVar(&projectSel, "project", "", "Project id or name (required)")
	cmd.Flags().StringVar(&envSel, "env", "", "Environment id or name (required)")
	cmd.Flags().StringVar(&dir, "dir", "", "Working directory (default: current directory)")
	cmd.Flags().StringSliceVar(&keys, "keys", nil, "Only request these secret keys (agent tokens only)")
	cmd.Flags().StringVar(&purpose, "purpose", "coding-agent", "Audit purpose for this secret request (agent tokens only)")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("env")

	return cmd
}

func withoutEnvKey(environment []string, key string) []string {
	prefix := key + "="
	filtered := make([]string, 0, len(environment))
	for _, entry := range environment {
		if len(entry) >= len(prefix) && entry[:len(prefix)] == prefix {
			continue
		}
		filtered = append(filtered, entry)
	}
	return filtered
}
