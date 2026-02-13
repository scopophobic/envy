package commands

import (
	"fmt"
	"os"

	"github.com/envo/cli/internal/config"
	"github.com/envo/cli/internal/store"
	"github.com/spf13/cobra"
)

type rootDeps struct {
	cfg    config.Config
	tokens *store.Tokens
}

func newRootCmd() (*cobra.Command, *rootDeps) {
	deps := &rootDeps{
		cfg:    config.Load(),
		tokens: nil,
	}

	cmd := &cobra.Command{
		Use:   "envo",
		Short: "Envo CLI",
	}

	cmd.PersistentFlags().String("api", "", "Envo API base URL (or set ENVO_API_URL)")

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if v, _ := cmd.Flags().GetString("api"); v != "" {
			deps.cfg.APIBaseURL = v
		}
		t, err := store.LoadTokens()
		if err != nil {
			return err
		}
		deps.tokens = t
		return nil
	}

	cmd.AddCommand(newLoginCmd(deps))
	cmd.AddCommand(newLogoutCmd())
	cmd.AddCommand(newWhoamiCmd(deps))
	cmd.AddCommand(newPullCmd(deps))
	cmd.AddCommand(newRunCmd(deps))

	return cmd, deps
}

func Execute() int {
	root, _ := newRootCmd()
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return 0
}

