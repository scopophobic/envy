package commands

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/envo/cli/internal/api"
	"github.com/envo/cli/internal/store"
	"github.com/spf13/cobra"
)

func newSyncCmd(deps *rootDeps) *cobra.Command {
	var (
		orgSel        string
		projectSel    string
		envSel        string
		connectionSel string
		targetProject string
		targetEnv     string
	)

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Manually sync an environment to a deploy platform",
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.tokens == nil {
				return fmt.Errorf("not logged in; run `envo login`")
			}
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

			connID, err := resolveConnectionID(ctx, client, connectionSel)
			if err != nil {
				return err
			}

			resp, err := client.SyncEnvironment(ctx, envID, api.SyncEnvironmentReq{
				PlatformConnectionID: connID,
				TargetProjectID:      strings.TrimSpace(targetProject),
				TargetEnvironment:    strings.TrimSpace(targetEnv),
			})
			if err != nil {
				return err
			}

			fmt.Printf("Synced %d secrets to %s (%s)\n", resp.Synced, resp.Platform, resp.ConnectionName)
			fmt.Printf("Target project: %s\n", resp.TargetProject)
			fmt.Printf("Target environment: %s\n", resp.TargetEnv)
			return nil
		},
	}

	cmd.Flags().StringVar(&orgSel, "org", "", "Organization id or name (required)")
	cmd.Flags().StringVar(&projectSel, "project", "", "Project id or name (required)")
	cmd.Flags().StringVar(&envSel, "env", "", "Environment id or name (required)")
	cmd.Flags().StringVar(&connectionSel, "connection", "", "Platform connection id or name (required)")
	cmd.Flags().StringVar(&targetProject, "target-project", "", "Remote deploy platform project ID (required)")
	cmd.Flags().StringVar(&targetEnv, "target-env", "", "Remote environment (development|preview|production)")
	_ = cmd.MarkFlagRequired("org")
	_ = cmd.MarkFlagRequired("project")
	_ = cmd.MarkFlagRequired("env")
	_ = cmd.MarkFlagRequired("connection")
	_ = cmd.MarkFlagRequired("target-project")
	_ = cmd.MarkFlagRequired("target-env")
	return cmd
}

func resolveConnectionID(ctx context.Context, c *api.Client, sel string) (string, error) {
	sel = strings.TrimSpace(sel)
	if sel == "" {
		return "", fmt.Errorf("--connection is required")
	}

	rows, err := c.ListPlatformConnections(ctx)
	if err != nil {
		return "", err
	}
	for _, row := range rows {
		if row.ID == sel {
			return row.ID, nil
		}
	}

	var matches []api.PlatformConnection
	for _, row := range rows {
		if strings.EqualFold(row.Name, sel) {
			matches = append(matches, row)
		}
	}
	if len(matches) == 1 {
		return matches[0].ID, nil
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple platform connections matched %q; use connection id", sel)
	}
	return "", fmt.Errorf("platform connection not found: %q", sel)
}
