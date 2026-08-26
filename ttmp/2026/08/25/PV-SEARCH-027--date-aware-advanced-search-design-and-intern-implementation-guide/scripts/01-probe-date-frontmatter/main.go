package main

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/go-go-golems/publish-vault/internal/parser"
)

type observation struct {
	Name     string      `json:"name"`
	Key      string      `json:"key"`
	GoType   string      `json:"go_type"`
	JSONType string      `json:"json_type"`
	Value    interface{} `json:"value"`
}

func main() {
	cases := []struct {
		name, key, value string
	}{
		{"date-only", "date", "2024-01-15"},
		{"created-date-only", "created", "2024-01-15"},
		{"updated-rfc3339", "updated", "2024-01-15T13:45:00-05:00"},
		{"quoted-date", "created", `"2024-01-15"`},
		{"invalid-date", "date", "January someday"},
	}
	out := make([]observation, 0, len(cases))
	for _, c := range cases {
		src := []byte(fmt.Sprintf("---\n%s: %s\n---\n# Probe\n", c.key, c.value))
		parsed, err := parser.Parse(src)
		if err != nil {
			panic(err)
		}
		v := parsed.Frontmatter[c.key]
		encoded, err := json.Marshal(v)
		if err != nil {
			panic(err)
		}
		var decoded interface{}
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			panic(err)
		}
		out = append(out, observation{
			Name: c.name, Key: c.key, GoType: reflect.TypeOf(v).String(),
			JSONType: reflect.TypeOf(decoded).String(), Value: v,
		})
	}
	encoded, err := json.MarshalIndent(map[string]interface{}{"schema_version": 1, "observations": out}, "", "  ")
	if err != nil {
		panic(err)
	}
	fmt.Println(string(encoded))
}
