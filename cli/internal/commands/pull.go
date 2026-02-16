package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/envo/cli/internal/api"
	"github.com/envo/cli/internal/dotenv"
	"github.com/envo/cli/internal/store"
	"github.com/spf13/cobra"
)

func newPullCmd(deps *rootDeps) *cobra.Command {
	var (
		orgSel     string
		projectSel string
		envSel     string
		outDir     string
	)

	cmd := &cobra.Command{
		Use:   "pull",
		Short: "Fetch secrets and write a .env file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.tokens == nil {
				return fmt.Errorf("not logged in; run `envo login`")
			}

			ctx, cancel := context.WithTimeout(cmd.Context(), 60*time.Second)
			defer cancel()

			client := api.NewClient(deps.cfg.APIBaseURL, deps.tokens)
			t, err := client.EnsureAccessToken(ctx)
			if err != nil {
				return err
			}
			// Persist refreshed access token
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

			if outDir == "" {
				// Prefer ENVO_CALLER_DIR (set by wrapper scripts that cd into cli/)
				if callerDir := os.Getenv("ENVO_CALLER_DIR"); callerDir != "" {
					outDir = callerDir
				} else {
					cwd, _ := os.Getwd()
					outDir = cwd
				}
			}
			outDir, _ = filepath.Abs(outDir)

			if err := dotenv.EnsureGitignoreHasDotenv(outDir); err != nil {
				return err
			}

			p, err := dotenv.WriteEnvFile(outDir, secrets)
			if err != nil {
				return err
			}

			fmt.Printf("Wrote %d secrets to %s\n", len(secrets), p)
			return nil
		},
	}

	cmd.Flags().StringVar(&orgSel, "org", "", "Organization id or name (required)")
	cmd.Flags().StringVar(&projectSel, "project", "", "Project id or name (required)")
	cmd.Flags().StringVar(&envSel, "env", "", "Environment id or name (required)")
	cmd.Flags().StringVar(&outDir, "dir", "", "Directory to write .env into (default: current directory)")
	_ = cmd.MarkFlagRequired("org")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("env")

	return cmd
}

func resolveOrgID(ctx context.Context, c *api.Client, sel string) (string, error) {
	sel = strings.TrimSpace(sel)
	if sel == "" {
		return "", fmt.Errorf("--org is required")
	}

	orgs, err := c.ListOrgs(ctx)
	if err != nil {
		return "", err
	}

	// direct id match
	for _, o := range orgs {
		if o.ID == sel {
			return o.ID, nil
		}
	}

	// case-insensitive name match
	var matches []api.Org
	for _, o := range orgs {
		if strings.EqualFold(o.Name, sel) {
			matches = append(matches, o)
		}
	}
	if len(matches) == 1 {
		return matches[0].ID, nil
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple orgs matched %q; use org id instead", sel)
	}

	return "", fmt.Errorf("org not found: %q", sel)
}

func resolveProjectID(ctx context.Context, c *api.Client, orgID string, sel string) (string, error) {
	sel = strings.TrimSpace(sel)
	if sel == "" {
		return "", fmt.Errorf("--project is required")
	}

	projects, err := c.ListOrgProjects(ctx, orgID)
	if err != nil {
		return "", err
	}

	for _, p := range projects {
		if p.ID == sel {
			return p.ID, nil
		}
	}

	var matches []api.Project
	for _, p := range projects {
		if strings.EqualFold(p.Name, sel) {
			matches = append(matches, p)
		}
	}
	if len(matches) == 1 {
		return matches[0].ID, nil
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple projects matched %q; use project id instead", sel)
	}

	return "", fmt.Errorf("project not found: %q", sel)
}

func resolveEnvID(ctx context.Context, c *api.Client, projectID string, sel string) (string, error) {
	sel = strings.TrimSpace(sel)
	if sel == "" {
		return "", fmt.Errorf("--env is required")
	}

	envs, err := c.ListProjectEnvironments(ctx, projectID)
	if err != nil {
		return "", err
	}

	for _, e := range envs {
		if e.ID == sel {
			return e.ID, nil
		}
	}

	var matches []api.Environment
	for _, e := range envs {
		if strings.EqualFold(e.Name, sel) {
			matches = append(matches, e)
		}
	}
	if len(matches) == 1 {
		return matches[0].ID, nil
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple environments matched %q; use environment id instead", sel)
	}

	return "", fmt.Errorf("environment not found: %q", sel)
}

