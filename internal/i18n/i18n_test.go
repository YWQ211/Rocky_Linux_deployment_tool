package i18n

import (
	"context"
	"errors"
	"reflect"
	"regexp"
	"testing"
)

func TestErrorPreservesCauseAndUserValues(t *testing.T) {
	err := Errorf("步骤 [%s] 失败: %w", "用户输入", context.Canceled)
	translated := LocalizeError("en", err)
	if !errors.Is(translated, context.Canceled) {
		t.Fatal("lost cause")
	}
	if translated.Error() != "Step [用户输入] failed: context canceled" {
		t.Fatal(translated)
	}
}
func TestCatalogFormatArguments(t *testing.T) {
	formats := regexp.MustCompile(`%[+#0 .0-9-]*[a-zA-Z]`)
	for key, row := range catalog {
		for _, lang := range []string{"en", "ja"} {
			got := Text(lang, key)
			if got == "" {
				t.Fatal(key, row)
			}
			if !reflect.DeepEqual(formats.FindAllString(key, -1), formats.FindAllString(got, -1)) {
				t.Fatalf("format mismatch: %q -> %q", key, got)
			}
		}
	}
}
