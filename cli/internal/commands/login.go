package commands

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"time"

	"github.com/envo/cli/internal/api"
	"github.com/envo/cli/internal/store"
	"github.com/spf13/cobra"
)

func newLoginCmd(deps *rootDeps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login via browser (Google OAuth)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Start localhost callback listener
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				return fmt.Errorf("failed to open localhost listener: %w", err)
			}
			defer ln.Close()

			addr := ln.Addr().String()
			callbackURL := "http://" + addr + "/callback"

			codeCh := make(chan string, 1)
			errCh := make(chan error, 1)

			mux := http.NewServeMux()
			mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
				q := r.URL.Query()
				code := q.Get("code")
				if code == "" {
					http.Error(w, "missing code", http.StatusBadRequest)
					return
				}
				fmt.Fprintln(w, "Envo CLI login complete. You can close this tab.")
				codeCh <- code
			})

			srv := &http.Server{Handler: mux}
			go func() {
				if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
					errCh <- err
				}
			}()
			defer func() {
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
				defer cancel()
				_ = srv.Shutdown(ctx)
			}()

			client := api.NewClient(deps.cfg.APIBaseURL, nil)
			startURL := client.CLIBrowserStartURL(callbackURL)

			fmt.Println("Opening browser for login...")
			fmt.Println("(Host in URL must match GOOGLE_REDIRECT_URL in backend .env — use --api http://127.0.0.1:8080 if you use 127.0.0.1 there.)")
			_ = openBrowser(startURL)
			fmt.Println("If it didn't open automatically, open this URL:")
			fmt.Println(startURL)

			// Wait for callback code, then exchange
			ctx, cancel := context.WithTimeout(cmd.Context(), 3*time.Minute)
			defer cancel()

			var code string
			select {
			case code = <-codeCh:
			case err := <-errCh:
				return err
			case <-ctx.Done():
				return fmt.Errorf("login timed out")
			}

			tokens, err := client.CLIExchange(cmd.Context(), code)
			if err != nil {
				return err
			}

			if err := store.SaveTokens(*tokens); err != nil {
				return err
			}

			fmt.Println("Login saved.")
			return nil
		},
	}

	return cmd
}

func openBrowser(u string) error {
	// validate URL early
	if _, err := url.Parse(u); err != nil {
		return err
	}

	switch runtime.GOOS {
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	case "darwin":
		return exec.Command("open", u).Start()
	default:
		return exec.Command("xdg-open", u).Start()
	}
}
