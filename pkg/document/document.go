// Package document provides utilities for working with multi-document YAML,
// as produced by tools like `helm template`. A single Helm release can render
// several Kubernetes resources (e.g. Deployment, Secret, Service) separated
// by YAML document markers (---). This package handles splitting, filtering,
// and inspecting those documents.
package document

import (
	"strings"

	"sigs.k8s.io/yaml"
)

// Split parses a multi-document YAML byte slice into individual documents.
// Documents are separated by lines containing only "---".
// Empty documents and those that parse to empty maps are silently skipped.
func Split(data []byte) ([]map[string]interface{}, error) {
	// Normalize line endings to \n for consistent splitting.
	content := strings.ReplaceAll(string(data), "\r\n", "\n")

	var docs []map[string]interface{}
	for _, part := range splitOnSeparator(content) {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		var doc map[string]interface{}
		if err := yaml.Unmarshal([]byte(part), &doc); err != nil {
			return nil, err
		}
		// Skip documents that parsed to nothing (e.g. comment-only sections).
		if len(doc) > 0 {
			docs = append(docs, doc)
		}
	}
	return docs, nil
}

// splitOnSeparator splits YAML content into segments on lines that are exactly "---".
// It correctly handles a leading "---" at the start of the file as well as
// separators embedded in the middle or end of the content.
func splitOnSeparator(content string) []string {
	var parts []string
	var current strings.Builder

	for _, line := range strings.Split(content, "\n") {
		if strings.TrimSpace(line) == "---" {
			// Flush the current document when we encounter a separator.
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
			continue
		}
		current.WriteString(line)
		current.WriteByte('\n')
	}
	// Flush any remaining content after the last separator.
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

// Match filters a list of documents by kind and/or metadata.name.
// Either filter can be left as an empty string to match any value.
// For example, Match(docs, "Deployment", "") returns all Deployments.
func Match(docs []map[string]interface{}, kind, name string) []map[string]interface{} {
	var result []map[string]interface{}
	for _, doc := range docs {
		// If a kind filter is set, only include docs with that kind.
		if kind != "" {
			if k, ok := doc["kind"].(string); !ok || k != kind {
				continue
			}
		}
		// If a name filter is set, only include docs with that metadata.name.
		if name != "" {
			meta, ok := doc["metadata"].(map[string]interface{})
			if !ok {
				continue
			}
			if n, ok := meta["name"].(string); !ok || n != name {
				continue
			}
		}
		result = append(result, doc)
	}
	return result
}

// Name returns the metadata.name field of a document, or an empty string if
// not present.
func Name(doc map[string]interface{}) string {
	if meta, ok := doc["metadata"].(map[string]interface{}); ok {
		if n, ok := meta["name"].(string); ok {
			return n
		}
	}
	return ""
}

// Kind returns the kind field of a document, or an empty string if not present.
func Kind(doc map[string]interface{}) string {
	if k, ok := doc["kind"].(string); ok {
		return k
	}
	return ""
}
