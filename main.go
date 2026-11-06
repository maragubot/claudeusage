// Command claudeusage prints the weekly Claude subscription usage, read with the OAuth credentials that Claude Code stores.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "claudeusage:", err)
		os.Exit(1)
	}
}

func run(args []string, w io.Writer) error {
	fs := flag.NewFlagSet("claudeusage", flag.ContinueOnError)
	raw := fs.Bool("json", false, "print the raw JSON response")
	if err := fs.Parse(args); err != nil {
		return err
	}

	token, err := readAccessToken(time.Now())
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	body, err := fetchUsage(ctx, token)
	if err != nil {
		return err
	}

	if *raw {
		_, err := fmt.Fprintln(w, strings.TrimSpace(string(body)))
		return err
	}

	u, err := parseUsage(body)
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(w, formatUsage(u, time.Now()))
	return err
}

// credentials as stored by Claude Code, in the login Keychain or in ~/.claude/.credentials.json.
type credentials struct {
	ClaudeAiOauth struct {
		AccessToken string `json:"accessToken"`
		ExpiresAt   int64  `json:"expiresAt"` // Unix milliseconds
	} `json:"claudeAiOauth"`
}

var errNoCredentials = errors.New(`no Claude Code credentials found in the Keychain (service "Claude Code-credentials") or in ~/.claude/.credentials.json; log in with Claude Code first`)

// readAccessToken from the login Keychain, falling back to ~/.claude/.credentials.json.
// It never refreshes the token and never writes anything back.
func readAccessToken(now time.Time) (string, error) {
	data, source, err := readCredentials()
	if err != nil {
		return "", err
	}
	return accessTokenFrom(data, source, now)
}

func readCredentials() ([]byte, string, error) {
	// The secret goes from the stdout of security straight into memory and is never echoed.
	cmd := exec.Command("/usr/bin/security", "find-generic-password", "-s", "Claude Code-credentials", "-w")
	if out, err := cmd.Output(); err == nil && len(strings.TrimSpace(string(out))) > 0 {
		return out, "the Keychain", nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, "", errNoCredentials
	}
	path := filepath.Join(home, ".claude", ".credentials.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", errNoCredentials
	}
	return data, path, nil
}

func accessTokenFrom(data []byte, source string, now time.Time) (string, error) {
	var c credentials
	if err := json.Unmarshal(data, &c); err != nil {
		// Don't wrap err: a JSON syntax error can quote part of the input.
		return "", fmt.Errorf("could not parse the Claude Code credentials in %v", source)
	}
	if c.ClaudeAiOauth.AccessToken == "" {
		return "", fmt.Errorf("the Claude Code credentials in %v have no OAuth access token; log in with Claude Code using your Claude subscription", source)
	}
	if c.ClaudeAiOauth.ExpiresAt > 0 && !now.Before(time.UnixMilli(c.ClaudeAiOauth.ExpiresAt)) {
		return "", fmt.Errorf("the Claude Code access token in %v expired at %v; run Claude Code once to refresh it, then try again",
			source, time.UnixMilli(c.ClaudeAiOauth.ExpiresAt).Local().Format("2006-01-02 15:04"))
	}
	return c.ClaudeAiOauth.AccessToken, nil
}

// fetchUsage from the OAuth usage endpoint. The token is only ever sent to api.anthropic.com over HTTPS.
func fetchUsage(ctx context.Context, token string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.anthropic.com/api/oauth/usage", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("anthropic-beta", "oauth-2025-04-20")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "claudeusage")

	client := &http.Client{
		// Never follow redirects, so the token can't be forwarded anywhere else.
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("could not reach the usage endpoint: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("could not read the usage response: %w", err)
	}

	if res.StatusCode != http.StatusOK {
		msg := strings.TrimSpace(strings.ReplaceAll(string(body), token, "[redacted]"))
		hint := ""
		if res.StatusCode == http.StatusUnauthorized {
			hint = " (run Claude Code once to refresh its login, then try again)"
		}
		return nil, fmt.Errorf("usage endpoint returned %v%v: %v", res.Status, hint, msg)
	}
	return body, nil
}
