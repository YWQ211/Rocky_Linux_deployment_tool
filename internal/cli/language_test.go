package cli

import (
	"context"
	"strings"
	"testing"
	"unicode"
)

func TestEnglishAndJapaneseCLI(t *testing.T) {
	for _, lang := range []string{"en", "ja"} {
		for _, cmd := range [][]string{{"--help"}, {"doctor"}, {"env", "install", "python", "--dry-run"}, {"mirror", "use", "tuna", "--dry-run"}, {"env", "list"}} {
			a, out := testApp("")
			args := append([]string{"--lang", lang}, cmd...)
			if err := a.Run(context.Background(), args); err != nil {
				t.Fatalf("%s %v: %v", lang, cmd, err)
			}
			if lang == "en" && strings.IndexFunc(out.String(), func(r rune) bool { return unicode.Is(unicode.Han, r) }) >= 0 {
				t.Fatalf("untranslated English output: %s", out.String())
			}
			if lang == "ja" && cmd[0] != "env" && !strings.ContainsAny(out.String(), "のをにしすル環境") {
				t.Fatalf("missing Japanese output: %s", out.String())
			}
		}
	}
}

func TestLanguageErrorsAndParameterPreservation(t *testing.T) {
	a, _ := testApp("")
	err := a.Run(context.Background(), []string{"--lang", "en", "env", "install", "用户输入"})
	if err == nil || !strings.Contains(err.Error(), "Unknown environment") || !strings.Contains(err.Error(), "用户输入") {
		t.Fatal(err)
	}
	if err = a.Run(context.Background(), []string{"--lang", "fr", "--help"}); err == nil {
		t.Fatal("unknown language accepted")
	}
}

func TestLanguageEnvironmentAndMenu(t *testing.T) {
	t.Setenv("ROCKDEV_LANG", "en")
	a, out := testApp("0\n")
	if err := a.Run(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "System check") {
		t.Fatal(out.String())
	}
	a, out = testApp("0\n")
	if err := a.Run(context.Background(), []string{"--lang=ja"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "システム確認") {
		t.Fatal(out.String())
	}
}
