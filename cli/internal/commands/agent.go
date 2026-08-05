package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/envo/cli/internal/api"
	"github.com/spf13/cobra"
)

func newAgentCmd(deps *rootDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "Inspect the agent identity provided through ENVO_TOKEN",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "whoami",
		Short: "Show the current Envo agent and credential",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if deps.cfg.AgentToken == "" {
				return fmt.Errorf("ENVO_TOKEN is not set")
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()
			me, err := api.NewAgentClient(deps.cfg.APIBaseURL, deps.cfg.AgentToken).GetAgentMe(ctx)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Agent: %s\nID: %s\nOrganization: %s\nCredential: %s (%s)\n", me.Agent.Name, me.Agent.ID, me.Agent.OrgID, me.CredentialName, me.CredentialID)
			return nil
		},
	})
	return cmd
}
