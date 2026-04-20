package email

import (
	"errors"
	"strings"
	"testing"
)

func TestBackoffFor(t *testing.T) {
	cases := []struct {
		attempt  int
		wantSecs int
		giveUp   bool
	}{
		{1, 1, false},
		{2, 4, false},
		{3, 16, false},
		{4, 64, false},
		{5, 0, true},
		{99, 0, true},
	}
	for _, c := range cases {
		gotSecs, gotGiveUp := BackoffFor(c.attempt)
		if gotSecs != c.wantSecs || gotGiveUp != c.giveUp {
			t.Errorf("BackoffFor(%d)=(%d,%t) want (%d,%t)", c.attempt, gotSecs, gotGiveUp, c.wantSecs, c.giveUp)
		}
	}
}

func TestIsRetryable(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"generic network", errors.New("dial tcp: timeout"), true},
		{"send 400 terminal", &SendError{StatusCode: 400, Retryable: false}, false},
		{"send 401 terminal", &SendError{StatusCode: 401, Retryable: false}, false},
		{"send 429 retryable", &SendError{StatusCode: 429, Retryable: true}, true},
		{"send 500 retryable", &SendError{StatusCode: 500, Retryable: true}, true},
		{"send 503 retryable", &SendError{StatusCode: 503, Retryable: true}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := IsRetryable(c.err); got != c.want {
				t.Errorf("IsRetryable(%v)=%t want %t", c.err, got, c.want)
			}
		})
	}
}

func TestNormalizeLang(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"", "ru"},
		{"ru", "ru"},
		{"RU", "ru"},
		{"en", "en"},
		{"  EN ", "en"},
		{"de", "ru"},   // unsupported → default
		{"kz", "ru"},   // unsupported → default
	}
	for _, c := range cases {
		if got := normalizeLang(c.in); got != c.want {
			t.Errorf("normalizeLang(%q)=%q want %q", c.in, got, c.want)
		}
	}
}

func TestRender_OTP_RU(t *testing.T) {
	subj, html, err := Render(TemplateOTP, "ru", map[string]any{
		"FirstName": "Иван",
		"Code":      "123456",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(subj, "код") && !strings.Contains(subj, "FoodBI") {
		t.Errorf("unexpected subject: %s", subj)
	}
	if !strings.Contains(html, "Иван") {
		t.Errorf("html missing FirstName: %s", html)
	}
	if !strings.Contains(html, "123456") {
		t.Errorf("html missing OTP code: %s", html)
	}
}

func TestRender_OTP_EN(t *testing.T) {
	subj, html, err := Render(TemplateOTP, "en", map[string]any{
		"FirstName": "John",
		"Code":      "654321",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(subj, "verification") {
		t.Errorf("unexpected EN subject: %s", subj)
	}
	if !strings.Contains(html, "654321") {
		t.Errorf("html missing code: %s", html)
	}
}

func TestRender_PasswordReset(t *testing.T) {
	_, html, err := Render(TemplatePasswordReset, "ru", map[string]any{
		"FirstName": "Пётр",
		"ResetURL":  "https://app.foodbi.local/reset?token=abc",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, "https://app.foodbi.local/reset?token=abc") {
		t.Errorf("html missing reset URL: %s", html)
	}
}

func TestRender_Invite(t *testing.T) {
	_, html, err := Render(TemplateInvite, "en", map[string]any{
		"CompanyName": "Acme Cafe",
		"Role":        "manager",
		"AcceptURL":   "https://app.foodbi.local/accept?token=xyz",
	})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(html, "Acme Cafe") || !strings.Contains(html, "manager") ||
		!strings.Contains(html, "https://app.foodbi.local/accept?token=xyz") {
		t.Errorf("html missing invite params: %s", html)
	}
}

func TestRender_UnknownTemplate(t *testing.T) {
	if _, _, err := Render("nope", "ru", nil); err == nil {
		t.Error("expected error for unknown template")
	}
}

func TestRender_UnknownLangFallsBack(t *testing.T) {
	// Unknown lang should fall back to ru silently.
	subj, _, err := Render(TemplateOTP, "fr", map[string]any{"FirstName": "X", "Code": "000000"})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// ru subject contains Cyrillic "код"
	if !strings.Contains(subj, "код") {
		t.Errorf("expected fallback to ru subject, got: %s", subj)
	}
}
