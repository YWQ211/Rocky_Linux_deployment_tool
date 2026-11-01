package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
)

//go:embed catalog.json
var files embed.FS
var catalog = func() map[string][2]string {
	raw, err := files.ReadFile("catalog.json")
	if err != nil {
		panic(err)
	}
	var rows [][3]string
	if err = json.Unmarshal(raw, &rows); err != nil {
		panic(err)
	}
	out := map[string][2]string{}
	for _, r := range rows {
		out[r[0]] = [2]string{r[1], r[2]}
	}
	return out
}()

func Text(lang, key string) string {
	if lang != "en" {
		for original, row := range catalog {
			if row[0] == key {
				if lang == "zh" {
					return original
				}
				if lang == "ja" {
					return row[1]
				}
			}
		}
	}
	if row, ok := catalog[key]; ok {
		if lang == "en" {
			return row[0]
		}
		if lang == "ja" {
			return row[1]
		}
	}
	return key
}

type messageError struct {
	format string
	args   []any
}

func Errorf(format string, args ...any) error { return &messageError{format, args} }
func (e *messageError) Error() string         { return fmt.Errorf(e.format, e.args...).Error() }
func (e *messageError) Unwrap() error {
	for _, arg := range e.args {
		if err, ok := arg.(error); ok {
			return err
		}
	}
	return nil
}

type translatedError struct {
	error
	message string
}

func (e translatedError) Error() string { return e.message }
func (e translatedError) Unwrap() error { return e.error }
func LocalizeError(lang string, err error) error {
	if err == nil {
		return err
	}
	if e, ok := err.(*messageError); ok {
		args := append([]any(nil), e.args...)
		for i, arg := range args {
			if child, ok := arg.(error); ok {
				args[i] = LocalizeError(lang, child)
			}
		}
		return translatedError{err, fmt.Errorf(Text(lang, e.format), args...).Error()}
	}
	return err
}
