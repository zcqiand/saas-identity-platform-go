// Command genopenapi — shared openapi.yaml → Swagger 文档面 openapi.json。
// servers 清空（同源 try-it-out 打本域，rails gen-shared.sh step 3 同款语义）。
// 零新依赖：yaml.v3 已在 go.mod（genmanifest 同源）。
package main

import (
	"encoding/json"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "usage: genopenapi <openapi.yaml> <openapi.json>")
		os.Exit(2)
	}
	raw, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, "read:", err)
		os.Exit(1)
	}
	var doc map[string]any
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		fmt.Fprintln(os.Stderr, "yaml:", err)
		os.Exit(1)
	}
	doc["servers"] = []any{}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "json:", err)
		os.Exit(1)
	}
	out = append(out, '\n')
	if err := os.WriteFile(os.Args[2], out, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "write:", err)
		os.Exit(1)
	}
}
