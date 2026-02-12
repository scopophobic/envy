package commands

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/envo/cli/internal/api"
	"github.com/envo/cli/internal/store"
	"github.com/spf13/cobra"
)

// Login UX note:
// Backend currently returns JSON tokens at the Google callback URL.
// For now, we ask the user to paste that JSON into the CLI.
func newLoginCmd(deps *rootDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login via Google OAuth",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(cmd.Context(), 30*time.Second)
			defer cancel()

			client := api.NewClient(deps.cfg.APIBaseURL, nil)
			u, err := client.GoogleLoginURL(ctx)
			if err != nil {
				return err
			}

			fmt.Println("Open this URL in your browser to authenticate:")
			fmt.Println(u)
			fmt.Println()
			fmt.Println("After login, your browser will show JSON containing access_token and refresh_token.")
			fmt.Println("Paste that JSON here, then press Enter:")

			reader := bufio.NewReader(os.Stdin)
			pasted, err := reader.ReadString('\n')
			if err != nil && strings.TrimSpace(pasted) == "" {
				return fmt.Errorf("failed to read pasted JSON: %w", err)
			}
			pasted = strings.TrimSpace(pasted)

			var parsed struct {
				AccessToken  string `json:"access_token"`
				RefreshToken string `json:"refresh_token"`
				TokenType    string `json:"token_type"`
				ExpiresIn    int    `json:"expires_in"`
			}
			if err := json.Unmarshal([]byte(pasted), &parsed); err != nil {
				return fmt.Errorf("invalid JSON; paste the single-line JSON response from the callback page")
			}

			if parsed.AccessToken == "" || parsed.RefreshToken == "" {
				return fmt.Errorf("missing tokens in pasted JSON")
			}

			t := store.Tokens{
				AccessToken:  parsed.AccessToken,
				RefreshToken: parsed.RefreshToken,
				TokenType:    parsed.TokenType,
				ExpiresAt:    time.Now().Add(time.Duration(parsed.ExpiresIn) * time.Second),
			}
			if t.TokenType == "" {
				t.TokenType = "Bearer"
			}

			if err := store.SaveTokens(t); err != nil {
				return err
			}

			fmt.Println("Login saved.")
			return nil
		},
	}

	return cmd
}

