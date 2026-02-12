package commands

import (
	"fmt"

	"github.com/envo/cli/internal/store"
	"github.com/spf13/cobra"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove locally cached credentials",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := store.ClearTokens(); err != nil {
				return err
			}
			fmt.Println("Logged out (local token cache removed).")
			return nil
		},
	}
}

