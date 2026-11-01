package core

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestProfileStrictValidation(t *testing.T) {
	good := `{"schema":1,"environments":["python","go"],"sources":[{"tool":"pip","provider":"tuna"}]}`
	p, err := DecodeProfile(strings.NewReader(good))
	if err != nil || len(p.Environments) != 2 {
		t.Fatal(p, err)
	}
	for _, bad := range []string{`{"schema":2}`, `{"schema":1,"shell":"touch /x"}`, `{"schema":1,"environments":["evil"]}`, `{"schema":1,"environments":["go","go"]}`, `{"schema":1,"update":false,"update":true}`, `{"schema":1,"environments":["python"],"update":false,"Update":true}`, `{"schema":1,"sources":[{"Tool":"npm","provider":"official"}]}`, `{"schema":1,"sources":[{"tool":"npm","provider":"bad"}]}`, good + ` {}`, `{"schema":1}`} {
		if _, err := DecodeProfile(strings.NewReader(bad)); err == nil {
			t.Fatal("accepted", bad)
		}
	}
}
func TestProfileExportRoundTrip(t *testing.T) {
	p := DeploymentProfile{Schema: 1, Environments: []string{"python"}, Venv: filepath.Join(t.TempDir(), "new-venv"), Sources: []ProfileSource{{Tool: "pip", Provider: "official"}}}
	raw, err := EncodeProfile(p)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeProfile(strings.NewReader(string(raw)))
	if err != nil || got.Venv != p.Venv || got.Update {
		t.Fatal(got, err)
	}
}
