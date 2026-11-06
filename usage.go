package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// usageResponse is the part of the usage endpoint response that claudeusage reads.
// The weekly windows are listed in limits; seven_day is a fallback for the all-models window.
type usageResponse struct {
	SevenDay *struct {
		Utilization *float64   `json:"utilization"`
		ResetsAt    *time.Time `json:"resets_at"`
	} `json:"seven_day"`
	Limits []struct {
		Kind     string     `json:"kind"`
		Group    string     `json:"group"`
		Percent  *float64   `json:"percent"`
		ResetsAt *time.Time `json:"resets_at"`
		Scope    *struct {
			Model *struct {
				DisplayName string `json:"display_name"`
			} `json:"model"`
		} `json:"scope"`
	} `json:"limits"`
}

// window of a usage limit. A nil window means the response doesn't report it.
type window struct {
	Percent  float64
	ResetsAt *time.Time
}

type usage struct {
	Weekly *window // all models
	Fable  *window
}

// parseUsage response body into the weekly all-models window and the weekly Fable window.
func parseUsage(body []byte) (usage, error) {
	var res usageResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return usage{}, fmt.Errorf("could not parse the usage response: %w", err)
	}

	var u usage
	for _, l := range res.Limits {
		if l.Group != "weekly" || l.Percent == nil {
			continue
		}
		w := &window{Percent: *l.Percent, ResetsAt: l.ResetsAt}
		switch {
		case l.Kind == "weekly_all" && l.Scope == nil:
			u.Weekly = w
		case l.Kind == "weekly_scoped" && l.Scope != nil && l.Scope.Model != nil &&
			strings.EqualFold(l.Scope.Model.DisplayName, "Fable"):
			u.Fable = w
		}
	}

	if u.Weekly == nil && res.SevenDay != nil && res.SevenDay.Utilization != nil {
		u.Weekly = &window{Percent: *res.SevenDay.Utilization, ResetsAt: res.SevenDay.ResetsAt}
	}

	if u.Weekly == nil && u.Fable == nil {
		return usage{}, errors.New("the usage response has no weekly limit; run with -json to see it")
	}
	return u, nil
}

// formatUsage as one line per weekly window, with the reset time in local time and relative to now.
func formatUsage(u usage, now time.Time) string {
	var b strings.Builder
	formatWindow(&b, "Weekly", u.Weekly, now)
	formatWindow(&b, "Fable", u.Fable, now)
	return b.String()
}

func formatWindow(b *strings.Builder, name string, w *window, now time.Time) {
	if w == nil {
		fmt.Fprintf(b, "%-6v   n/a\n", name)
		return
	}
	fmt.Fprintf(b, "%-6v  %3.0f%%", name, w.Percent)
	if w.ResetsAt != nil {
		fmt.Fprintf(b, "  resets %v (%v)", w.ResetsAt.In(now.Location()).Format("Mon 2 Jan 15:04"), relative(w.ResetsAt.Sub(now)))
	}
	b.WriteString("\n")
}

// relative duration until a reset, like "in 3h12m" or "in 2d4h".
func relative(d time.Duration) string {
	if d <= 0 {
		return "now"
	}
	d = d.Round(time.Minute)
	days := int(d / (24 * time.Hour))
	hours := int(d % (24 * time.Hour) / time.Hour)
	minutes := int(d % time.Hour / time.Minute)
	switch {
	case days > 0:
		return fmt.Sprintf("in %vd%vh", days, hours)
	case hours > 0:
		return fmt.Sprintf("in %vh%vm", hours, minutes)
	default:
		return fmt.Sprintf("in %vm", minutes)
	}
}
