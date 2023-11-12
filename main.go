package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	diff "github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"
	"sigs.k8s.io/yaml"
)

type APISchema struct {
	Openapi    string                `json:"openapi"`
	Info       Info                  `json:"info"`
	Paths      map[string]APIMethods `json:"paths"`
	Components *Components           `json:"components,omitempty"`
}

type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

type APIMethods map[string]APIMethod

type APIMethod struct {
	Summary     string            `json:"summary"`
	OperationId string            `json:"operationId,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	RequestBody json.RawMessage   `json:"requestBody,omitempty"`
	Parameters  []json.RawMessage `json:"parameters,omitempty"`
	Responses   json.RawMessage   `json:"responses"`
}

type Components struct {
	Schemas map[string]json.RawMessage `json:"schemas"`
}

func main() {
	dir := "openapi"

	err := checkParsing(dir)
	if err != nil {
		panic(err)
	}

	schema, err := combine(dir)
	if err != nil {
		panic(err)
	}

	data, err := yaml.Marshal(schema)
	if err != nil {
		panic(err)
	}
	err = os.WriteFile("api.yaml", data, 0644)
	if err != nil {
		panic(err)
	}
}

func checkParsing(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}

		filename := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filename)
		if err != nil {
			return err
		}
		jsonOriginal, err := yaml.YAMLToJSON(data)
		if err != nil {
			return err
		}
		var original map[string]interface{}
		err = json.Unmarshal(jsonOriginal, &original)
		if err != nil {
			return errors.Wrap(err, filename)
		}

		var schema APISchema
		err = yaml.Unmarshal(data, &schema)
		if err != nil {
			return err
		}

		jsonParsed, err := json.Marshal(schema)
		if err != nil {
			return err
		}

		// Then, Check them
		differ := diff.New()
		d, err := differ.Compare(jsonOriginal, jsonParsed)
		if err != nil {
			return errors.Wrap(err, filename)
		}

		if d.Modified() {
			config := formatter.AsciiFormatterConfig{
				ShowArrayIndex: true,
				Coloring:       true,
			}

			f := formatter.NewAsciiFormatter(original, config)
			result, err := f.Format(d)
			if err != nil {
				return errors.Wrap(err, filename)
			}
			fmt.Println(result)
			return errors.Errorf("%s schema is not preserved", filename)
		}
	}
	return nil
}

func combine(dir string) (*APISchema, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	combined := APISchema{
		Openapi: "3.0.1",
		Info: Info{
			Title:   "Apache JAMES Web Admin API",
			Version: "3.8.0",
		},
		Paths: map[string]APIMethods{},
		Components: &Components{
			Schemas: map[string]json.RawMessage{},
		},
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}

		filename := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(filename)
		if err != nil {
			return nil, err
		}
		var schema APISchema
		err = yaml.Unmarshal(data, &schema)
		if err != nil {
			return nil, err
		}

		for path, methods := range schema.Paths {
			for method, schema := range methods {
				tag, _, found := strings.Cut(entry.Name(), ".")
				if found {
					schema.Tags = []string{tag}
				}
				methods[method] = schema
			}
			schema.Paths[path] = methods

			combined.Paths[path] = methods
		}

		if schema.Components != nil {
			for msg, schema := range schema.Components.Schemas {
				_, found := combined.Components.Schemas[msg]
				if found {
					return nil, errors.Errorf("duplicate message %s in file %s", msg, filename)
				}
				combined.Components.Schemas[msg] = schema
			}
		}

	}

	return &combined, nil
}
