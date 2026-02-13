package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/envo/cli/internal/api"
	"github.com/spf13/cobra"
)

func newWhoamiCmd(deps *rootDeps) *cobra.Command {
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show current logged-in user",
		RunE: func(cmd *cobra.Command, args []string) error {
			if deps.tokens == nil {
				fmt.Println("Not logged in. Run: envo login")
				return nil
			}
			ctx, cancel := context.WithTimeout(cmd.Context(), 15*time.Second)
			defer cancel()
			client := api.NewClient(deps.cfg.APIBaseURL, deps.tokens)
			user, err := client.GetCurrentUser(ctx)
			if err != nil {
				return err
			}
			fmt.Println("Logged in as:")
			fmt.Printf("  Name:   %s\n", user.Name)
			fmt.Printf("  Email:  %s\n", user.Email)
			fmt.Printf("  ID:     %s\n", user.ID)
			fmt.Printf("  Tier:   %s\n", user.Tier)
			return nil
		},
	}
}
