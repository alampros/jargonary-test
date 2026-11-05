package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestSchemaVersion(t *testing.T) {
	for _, configuration := range []bool{false, true} {
		for _, version := range []string{"", "v1", "1.2.3", "v1.2.3/other"} {
			var out bytes.Buffer
			if err := writeSchema(&out, version, configuration); err == nil {
				t.Errorf("version %q should fail", version)
			}
			if out.Len() != 0 {
				t.Error("invalid version wrote schema output")
			}
		}
		for _, version := range []string{"v0.0.0", "v1.2.3", "v2.0.0-rc.1"} {
			var out bytes.Buffer
			if err := writeSchema(&out, version, configuration); err != nil {
				t.Fatal(err)
			}
			var schema struct {
				ID string `json:"$id"`
			}
			if err := json.Unmarshal(out.Bytes(), &schema); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(schema.ID, "/schema/"+version+"/") {
				t.Errorf("schema ID %q does not use supplied version %q", schema.ID, version)
			}
		}
	}
}

func TestGeneratedSchemas(t *testing.T) {
	manifestJSON, err := os.ReadFile("../../.release-please-manifest.json")
	if err != nil {
		t.Fatal(err)
	}
	var manifest map[string]string
	if err := json.Unmarshal(manifestJSON, &manifest); err != nil {
		t.Fatal(err)
	}
	version := "v" + manifest["."]
	for _, tt := range []struct {
		file          string
		configuration bool
	}{{"config.schema.json", true}, {"glossary.schema.json", false}} {
		t.Run(tt.file, func(t *testing.T) {
			var want bytes.Buffer
			if err := writeSchema(&want, version, tt.configuration); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile("../../docs-site/public/schema/" + version + "/" + tt.file)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, want.Bytes()) {
				t.Errorf("schema is stale; run go tool task schema VERSION=%s", version)
			}
		})
	}
}

func TestPartOfSpeechSchemaEnum(t *testing.T) {
	var out bytes.Buffer
	if err := writeSchema(&out, "v0.1.2", false); err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Defs map[string]struct {
			Properties map[string]struct {
				Enum []string `json:"enum"`
			} `json:"properties"`
			Required []string `json:"required"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(out.Bytes(), &schema); err != nil {
		t.Fatal(err)
	}
	want := "noun,pronoun,verb,adjective,adverb,preposition,conjunction,interjection,determiner"
	if got := strings.Join(schema.Defs["Sense"].Properties["partOfSpeech"].Enum, ","); got != want {
		t.Fatalf("partOfSpeech enum = %q, want %q", got, want)
	}
	for _, field := range schema.Defs["Sense"].Required {
		if field == "partOfSpeech" {
			t.Fatal("partOfSpeech must remain optional")
		}
	}
}
