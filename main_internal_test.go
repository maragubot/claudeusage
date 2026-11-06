package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestAccessTokenFrom(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)

	t.Run("should return the access token when it has not expired", func(t *testing.T) {
		data := []byte(`{"claudeAiOauth":{"accessToken":"fake-token","refreshToken":"fake-refresh","expiresAt":` +
			fmtMillis(now.Add(time.Hour)) + `}}`)
		token, err := accessTokenFrom(data, "test", now)
		if err != nil {
			t.Fatal(err)
		}
		if token != "fake-token" {
			t.Fatalf("expected the access token, got %v", token)
		}
	})

	t.Run("should return an error telling the user to run Claude Code when the token has expired", func(t *testing.T) {
		data := []byte(`{"claudeAiOauth":{"accessToken":"fake-token","expiresAt":` + fmtMillis(now.Add(-time.Minute)) + `}}`)
		_, err := accessTokenFrom(data, "test", now)
		if err == nil || !strings.Contains(err.Error(), "run Claude Code once") {
			t.Fatalf("expected an expiry error, got %v", err)
		}
		if strings.Contains(err.Error(), "fake-token") {
			t.Fatal("error must not contain the token")
		}
	})

	t.Run("should return an error when there is no access token", func(t *testing.T) {
		if _, err := accessTokenFrom([]byte(`{"claudeAiOauth":{}}`), "test", now); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("should return an error without the input when the credentials are invalid JSON", func(t *testing.T) {
		_, err := accessTokenFrom([]byte(`{"claudeAiOauth":{"accessToken":"fake-token"`), "test", now)
		if err == nil {
			t.Fatal("expected an error")
		}
		if strings.Contains(err.Error(), "fake-token") {
			t.Fatal("error must not contain the token")
		}
	})
}

func fmtMillis(t time.Time) string {
	return fmt.Sprint(t.UnixMilli())
}
