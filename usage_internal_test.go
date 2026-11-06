package main

import (
	"os"
	"testing"
	"time"
)

func TestParseUsage(t *testing.T) {
	t.Run("should read the weekly all-models and the weekly Fable windows from the limits", func(t *testing.T) {
		body, err := os.ReadFile("testdata/usage.json")
		if err != nil {
			t.Fatal(err)
		}

		u, err := parseUsage(body)
		if err != nil {
			t.Fatal(err)
		}

		if u.Weekly == nil || u.Weekly.Percent != 67 {
			t.Fatalf("expected weekly 67%%, got %+v", u.Weekly)
		}
		if u.Fable == nil || u.Fable.Percent != 100 {
			t.Fatalf("expected Fable 100%%, got %+v", u.Fable)
		}
		expected := time.Date(2026, 11, 8, 19, 59, 59, 660010000, time.UTC)
		if u.Fable.ResetsAt == nil || !u.Fable.ResetsAt.Equal(expected) {
			t.Fatalf("expected Fable reset %v, got %v", expected, u.Fable.ResetsAt)
		}
	})

	t.Run("should fall back to seven_day when the limits are missing, and leave Fable nil", func(t *testing.T) {
		u, err := parseUsage([]byte(`{"five_hour":null,"seven_day":{"utilization":12.5,"resets_at":"2026-11-06T19:59:59+00:00"}}`))
		if err != nil {
			t.Fatal(err)
		}
		if u.Weekly == nil || u.Weekly.Percent != 12.5 {
			t.Fatalf("expected weekly 12.5%%, got %+v", u.Weekly)
		}
		if u.Fable != nil {
			t.Fatalf("expected no Fable window, got %+v", u.Fable)
		}
	})

	t.Run("should ignore scoped limits for other models", func(t *testing.T) {
		u, err := parseUsage([]byte(`{"limits":[
			{"kind":"weekly_all","group":"weekly","percent":5,"resets_at":null,"scope":null},
			{"kind":"weekly_scoped","group":"weekly","percent":90,"scope":{"model":{"display_name":"Sonnet"}}}
		]}`))
		if err != nil {
			t.Fatal(err)
		}
		if u.Fable != nil {
			t.Fatalf("expected no Fable window, got %+v", u.Fable)
		}
	})

	t.Run("should return an error when there is no weekly limit", func(t *testing.T) {
		if _, err := parseUsage([]byte(`{"five_hour":null,"seven_day":null,"limits":[]}`)); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("should return an error for invalid JSON", func(t *testing.T) {
		if _, err := parseUsage([]byte(`<html>`)); err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestFormatUsage(t *testing.T) {
	t.Run("should print one line per weekly window with local and relative reset times", func(t *testing.T) {
		body, err := os.ReadFile("testdata/usage.json")
		if err != nil {
			t.Fatal(err)
		}
		u, err := parseUsage(body)
		if err != nil {
			t.Fatal(err)
		}

		cet := time.FixedZone("CET", 60*60)
		now := time.Date(2026, 11, 6, 9, 48, 0, 0, time.UTC).In(cet)

		expected := "Weekly   67%  resets Fri 6 Nov 20:59 (in 10h12m)\n" +
			"Fable   100%  resets Sun 8 Nov 20:59 (in 2d10h)\n"
		if got := formatUsage(u, now); got != expected {
			t.Fatalf("expected\n%v\ngot\n%v", expected, got)
		}
	})

	t.Run("should print n/a for a missing window and skip a missing reset time", func(t *testing.T) {
		got := formatUsage(usage{Weekly: &window{Percent: 5}}, time.Now())
		expected := "Weekly    5%\nFable    n/a\n"
		if got != expected {
			t.Fatalf("expected\n%v\ngot\n%v", expected, got)
		}
	})
}

func TestRelative(t *testing.T) {
	tests := []struct {
		name     string
		d        time.Duration
		expected string
	}{
		{name: "should say now for a reset in the past", d: -time.Minute, expected: "now"},
		{name: "should show minutes only under an hour", d: 42 * time.Minute, expected: "in 42m"},
		{name: "should show hours and minutes under a day", d: 3*time.Hour + 12*time.Minute + 20*time.Second, expected: "in 3h12m"},
		{name: "should show days and hours from a day", d: 50*time.Hour + 30*time.Minute, expected: "in 2d2h"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := relative(test.d); got != test.expected {
				t.Fatalf("expected %v, got %v", test.expected, got)
			}
		})
	}
}
