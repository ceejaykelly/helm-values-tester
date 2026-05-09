# Helm Values Tester (hvt)

A CLI tool to merge YAML values and assert conditions on the result, designed for Helm workflows and CI pipelines.

## Features
- Merge multiple YAML files (base + overrides)
- Pipe YAML (e.g., from `helm template`) into hvt
- Assert values in the merged YAML using inline or file-based assertions

## Installation

Build from source:
```sh
go build -o hvt ./cmd/hvt
```

## Usage

```
hvt <file1.yaml> [file2.yaml ...] [--assert key.path==value] [--assert-file asserts.yaml]
```

- Use `-` to read YAML from stdin (e.g., piped from `helm template`).

### Options

- `file1.yaml [file2.yaml ...]`  
  One or more YAML files to merge (first is base, rest are overrides).

- `-`  
  Read YAML from stdin.

- `--assert <path><operator><value>`  
  Inline assertion. Example:  
  `--assert spec.template.spec.containers[0].image==nginx:latest`

- `--assert-file <asserts.yaml>`  
  Load multiple assertions from a YAML file. Example file:
  ```yaml
  - path: spec.template.metadata.labels.app
    operator: ==
    expected: my-app
  - path: spec.replicas
    operator: ==
    expected: 3
  ```

### Operators

- `==`  (equal)
- `!=`  (not equal)
- (future: `>`, `<`, `>=`, `<=`)

### Examples

**Merge two files and assert:**
```sh
hvt base.yaml override.yaml --assert foo.bar==baz
```

**Pipe Helm output and assert:**
```sh
helm template myrelease ./mychart | hvt --assert spec.replicas==3
```

**Use an assertion file:**
```sh
hvt values.yaml --assert-file asserts.yaml
```

---

Contributions and issues welcome!
