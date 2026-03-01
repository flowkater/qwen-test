package domain

import "testing"

func TestResolveAPIKeyPriority(t *testing.T) {
	key, source, err := ResolveAPIKey("", map[string]string{"OPENROUTER_API_KEY": "env-key"})
	if err != nil || key != "env-key" || source != "env" {
		t.Fatalf("expected env key, got key=%q source=%q err=%v", key, source, err)
	}
	key, source, err = ResolveAPIKey("flag-key", map[string]string{"OPENROUTER_API_KEY": "env-key"})
	if err != nil || key != "flag-key" || source != "flag" {
		t.Fatalf("expected flag key, got key=%q source=%q err=%v", key, source, err)
	}
}

func TestResolveAPIKeyMissing(t *testing.T) {
	_, _, err := ResolveAPIKey("", map[string]string{})
	if err == nil {
		t.Fatalf("expected error")
	}
}

func TestMaskSecret(t *testing.T) {
	if got := MaskSecret("abcdef123456"); got != "abc...456" {
		t.Fatalf("unexpected masked value: %s", got)
	}
}
