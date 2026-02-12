package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/envo/cli/internal/api"
	"github.com/envo/cli/internal/dotenv"
	"github.com/envo/cli/internal/store"
	"github.com/spf13/cobra"
)

func newRunCmd(deps *rootDeps) *cobra.Command {
	var (
		orgSel     string
		projectSel string
		envSel     string
		dir        string
	)

	cmd := &cobra.Command{
		Use:   "run --org <org> --project <project> --env <env> -- <command> [args...]",
		Short: "Pull secrets then run a command with them",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.tokens == nil {
				return fmt.Errorf("not logged in; run `envo login`")
			}

			if dir == "" {
				cwd, _ := os.Getwd()
				dir = cwd
			}
			dir, _ = filepath.Abs(dir)

			ctx, cancel := context.WithTimeout(cmd.Context(), 90*time.Second)
			defer cancel()

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

			secrets, err := client.ExportEnvironmentSecrets(ctx, envID)
			if err != nil {
				return err
			}

			if err := dotenv.EnsureGitignoreHasDotenv(dir); err != nil {
				return err
			}
			envPath, err := dotenv.WriteEnvFile(dir, secrets)
			if err != nil {
				return err
			}

			envMap, err := dotenv.LoadEnvFile(envPath)
			if err != nil {
				return err
			}

			child := exec.CommandContext(cmd.Context(), args[0], args[1:]...)
			child.Dir = dir
			child.Stdout = os.Stdout
			child.Stderr = os.Stderr
			child.Stdin = os.Stdin
			child.Env = os.Environ()
			for k, v := range envMap {
				child.Env = append(child.Env, k+"="+v)
			}

			return child.Run()
		},
	}

	cmd.Flags().StringVar(&orgSel, "org", "", "Organization id or name (required)")
	cmd.Flags().StringVar(&projectSel, "project", "", "Project id or name (required)")
	cmd.Flags().StringVar(&envSel, "env", "", "Environment id or name (required)")
	cmd.Flags().StringVar(&dir, "dir", "", "Working directory / where to write .env (default: current directory)")
	_ = cmd.MarkFlagRequired("org")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("env")

	return cmd
}

