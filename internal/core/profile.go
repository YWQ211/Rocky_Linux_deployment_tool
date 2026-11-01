package core

import (
	"bytes"
	"encoding/json"
	"io"
	"path/filepath"
	"rockdev/internal/i18n"
	"strings"
)

type ProfileSource struct {
	Tool     string `json:"tool"`
	Provider string `json:"provider"`
}
type DeploymentProfile struct {
	Schema       int             `json:"schema"`
	Environments []string        `json:"environments,omitempty"`
	Update       bool            `json:"update,omitempty"`
	Venv         string          `json:"venv,omitempty"`
	Sources      []ProfileSource `json:"sources,omitempty"`
}

func DecodeProfile(r io.Reader) (DeploymentProfile, error) {
	data, err := io.ReadAll(io.LimitReader(r, (1<<20)+1))
	if err != nil {
		return DeploymentProfile{}, err
	}
	if len(data) > 1<<20 {
		return DeploymentProfile{}, i18n.Errorf("Profile exceeds 1 MiB")
	}
	if err = rejectDuplicateKeys(json.NewDecoder(bytes.NewReader(data))); err != nil {
		return DeploymentProfile{}, err
	}
	var p DeploymentProfile
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err = dec.Decode(&p); err != nil {
		return p, err
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return p, i18n.Errorf("Profile must contain one JSON object")
	}
	return p, p.Validate()
}
func (p DeploymentProfile) Validate() error {
	if p.Schema != 1 {
		return i18n.Errorf("Unsupported profile schema %d", p.Schema)
	}
	if len(p.Environments) == 0 && len(p.Sources) == 0 {
		return i18n.Errorf("Profile has no deployment tasks")
	}
	known := map[string]bool{}
	for _, e := range Environments {
		known[e.ID] = true
	}
	seen := map[string]bool{}
	for _, e := range p.Environments {
		if !known[e] || seen[e] {
			return i18n.Errorf("Unknown or duplicate environment %q", e)
		}
		seen[e] = true
	}
	if p.Venv != "" && (!filepath.IsAbs(p.Venv) || !seen["python"]) {
		return i18n.Errorf("Profile venv requires Python and an absolute path")
	}
	tools := map[string]bool{}
	for _, s := range p.Sources {
		if tools[s.Tool] {
			return i18n.Errorf("Duplicate source tool %q", s.Tool)
		}
		tools[s.Tool] = true
		providers, err := SourceProviders(s.Tool)
		if err != nil {
			return err
		}
		found := false
		for _, v := range providers {
			if v.ID == s.Provider {
				found = true
			}
		}
		if !found {
			return i18n.Errorf("Unknown source provider %q for %s", s.Provider, s.Tool)
		}
	}
	return nil
}
func EncodeProfile(p DeploymentProfile) ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(p, "", "  ")
	return append(data, '\n'), err
}

// The ordinary JSON decoder accepts duplicate keys; deployment policy must not.
func rejectDuplicateKeys(dec *json.Decoder) error { return rejectProfileKeys(dec, 0) }
func rejectProfileKeys(dec *json.Decoder, depth int) error {
	if depth > 8 {
		return i18n.Errorf("Profile nesting is too deep")
	}
	token, err := dec.Token()
	if err != nil {
		return err
	}
	delimiter, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delimiter {
	case '{':
		seen := map[string]bool{}
		for dec.More() {
			key, err := dec.Token()
			if err != nil {
				return err
			}
			s, ok := key.(string)
			if ok && s != strings.ToLower(s) {
				return i18n.Errorf("Profile keys must use lowercase: %s", s)
			}
			if !ok || seen[s] {
				return i18n.Errorf("Duplicate JSON key %q", key)
			}
			seen[s] = true
			if err = rejectProfileKeys(dec, depth+1); err != nil {
				return err
			}
		}
		_, err = dec.Token()
		return err
	case '[':
		for dec.More() {
			if err = rejectProfileKeys(dec, depth+1); err != nil {
				return err
			}
		}
		_, err = dec.Token()
		return err
	default:
		return i18n.Errorf("Invalid JSON delimiter")
	}
}
