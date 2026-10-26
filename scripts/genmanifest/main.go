// scripts/genmanifest/main.go — OpenAPI -> API manifest 生成器（REQ-2026-003）
//
// openapi-generator 对 go 无自研 manifest 先例必要——本工具镜像 rails
// scripts/gen-manifest.rb / laravel scripts/gen-manifest.php 同语义：
// 读 shared openapi.yaml，emit generated/api_manifest.json：
//
//	{ "schema": 1, "operations": [ { operationId, method, path, tag,
//	    requestSchema, responseSchema, requiredFields } ] }
//
// 确定性输出（L4.codegen.idempotent compares regen bytes）：
// - operations 按 [tag, operationId] 排序；键序 = 结构体字段序固定
// - 无时间戳；LF 写盘（禁 CRLF）；UTF-8
//
// 用法：go run ./scripts/genmanifest <openapi.yaml> [--out DIR]（默认 generated）
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const defaultOut = "generated"

var httpMethods = map[string]bool{"get": true, "post": true, "put": true, "patch": true, "delete": true}

type operation struct {
	OperationID    string   `json:"operationId"`
	Method         string   `json:"method"`
	Path           string   `json:"path"`
	Tag            string   `json:"tag"`
	RequestSchema  *string  `json:"requestSchema"`
	ResponseSchema *string  `json:"responseSchema"`
	RequiredFields []string `json:"requiredFields"`
}

type manifest struct {
	Schema     int         `json:"schema"`
	Operations []operation `json:"operations"`
}

func main() {
	args := os.Args[1:]
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "usage: genmanifest <openapi.yaml> [--out DIR]")
		os.Exit(1)
	}
	openapiPath := args[0]
	outDir := defaultOut
	for i := 1; i < len(args); i++ {
		if args[i] == "--out" {
			if i+1 >= len(args) {
				fmt.Fprintln(os.Stderr, "--out needs a directory argument")
				os.Exit(1)
			}
			i++
			outDir = args[i]
		}
	}

	raw, err := os.ReadFile(openapiPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: read %s: %v\n", openapiPath, err)
		os.Exit(1)
	}
	var spec map[string]interface{}
	if err := yaml.Unmarshal(raw, &spec); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: parse yaml: %v\n", err)
		os.Exit(1)
	}
	paths, ok := spec["paths"].(map[string]interface{})
	if !ok {
		fmt.Fprintf(os.Stderr, "ERROR: %s missing paths\n", openapiPath)
		os.Exit(1)
	}
	components, _ := spec["components"].(map[string]interface{})
	schemas, _ := components["schemas"].(map[string]interface{})

	refTail := func(node interface{}) *string {
		m, ok := node.(map[string]interface{})
		if !ok {
			return nil
		}
		ref, ok := m["$ref"].(string)
		if !ok || ref == "" {
			return nil
		}
		parts := strings.Split(ref, "/")
		tail := parts[len(parts)-1]
		return &tail
	}

	var ops []operation
	for path, item := range paths {
		pathItem, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		for method, opNode := range pathItem {
			if !httpMethods[method] {
				continue
			}
			op, ok := opNode.(map[string]interface{})
			if !ok {
				continue
			}
			opID, _ := op["operationId"].(string)
			if opID == "" {
				fmt.Fprintf(os.Stderr, "ERROR: %s %s missing operationId\n", path, method)
				os.Exit(1)
			}
			tags, _ := op["tags"].([]interface{})
			tag := ""
			if len(tags) > 0 {
				tag, _ = tags[0].(string)
			}
			if tag == "" {
				fmt.Fprintf(os.Stderr, "ERROR: %s %s missing tags.first\n", path, method)
				os.Exit(1)
			}

			requestSchema := refTail(mapGet(op, "requestBody", "content", "application/json", "schema"))
			resp, _ := op["responses"].(map[string]interface{})
			responseSchema := refTail(mapGet(resp, "200", "content", "application/json", "schema"))

			requiredFields := []string{}
			if requestSchema != nil {
				if schemaNode, ok := schemas[*requestSchema].(map[string]interface{}); ok {
					if req, ok := schemaNode["required"].([]interface{}); ok {
						for _, f := range req {
							if s, ok := f.(string); ok {
								requiredFields = append(requiredFields, s)
							}
						}
					}
				}
			}

			ops = append(ops, operation{
				OperationID:    opID,
				Method:         strings.ToUpper(method),
				Path:           path,
				Tag:            tag,
				RequestSchema:  requestSchema,
				ResponseSchema: responseSchema,
				RequiredFields: requiredFields,
			})
		}
	}

	sort.SliceStable(ops, func(i, j int) bool {
		if ops[i].Tag != ops[j].Tag {
			return ops[i].Tag < ops[j].Tag
		}
		return ops[i].OperationID < ops[j].OperationID
	})

	out := manifest{Schema: 1, Operations: ops}
	jsonBytes, err := json.MarshalIndent(out, "", "    ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: json encode: %v\n", err)
		os.Exit(1)
	}
	jsonBytes = append(jsonBytes, '\n')
	if strings.ContainsRune(string(jsonBytes), '\r') {
		fmt.Fprintln(os.Stderr, "ERROR: CR byte in output, refusing to write (determinism)")
		os.Exit(1)
	}

	outPath := filepath.Join(strings.TrimRight(outDir, `/\`), "api_manifest.json")
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: mkdir: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(outPath, jsonBytes, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: write failed: %s: %v\n", outPath, err)
		os.Exit(1)
	}
	fmt.Printf("[gen-manifest] %d operations -> %s\n", len(ops), outPath)
}

// mapGet 沿键路径取嵌套节点，任一层缺失/非 map 返回 nil（镜像 php ?? null 链）。
func mapGet(node interface{}, keys ...string) interface{} {
	cur := node
	for _, k := range keys {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil
		}
		cur = m[k]
	}
	return cur
}
