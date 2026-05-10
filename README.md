# YAML Assertions (ya)

A CLI tool to merge Helm values files and assert conditions on rendered YAML output, designed for Helm workflows and CI pipelines.

## Features
- Merge multiple YAML values files with Helm-aware coalescing (base + overrides)
- Pipe multi-document YAML (e.g., from `helm template`) directly into `ya`
- Assert values across multiple Kubernetes resources using inline or file-based assertions
- Filter assertions by Kubernetes resource `kind` and `metadata.name`
- Colored, leveled output (`PASS`/`FAIL`/`ERROR`) with correct exit codes for CI

## Installation

Build from source:
```sh
go build -o ya ./cmd/ya
```

## Usage

`ya` has two subcommands: `merge` and `assert`.

### merge

Merge two or more YAML values files in order (first is base, rest are overrides) and print the result:

```sh
ya merge base.yaml override.yaml [more.yaml ...]
```

### assert

Assert conditions on a multi-document YAML stream. Input is read from files or stdin.

```sh
ya assert [--assert <expr>] [--assert-file <file>] [file.yaml ... | -]
```

- Omit files (or pass `-`) to read from stdin.
- Returns exit code `0` if all assertions pass, `1` if any fail.

#### Flags

- `--assert <path><operator><value>`  
  Inline assertion. May be repeated. Example:  
  ```sh
  --assert spec.template.spec.containers[0].image==nginx:latest
  ```

- `--assert-file <asserts.yaml>`  
  Load assertions from a YAML file. May be repeated.

#### Assertion file format

Assert files are **named maps** — each key is a unique assertion name. This design makes files mergeable: if you pass multiple `--assert-file` flags, assertions are overlaid in order, with later files overriding keys from earlier files (useful for base + environment-specific suites).

```yaml
check-replicas:
  kind: Deployment          # optional: filter by resource kind
  name: my-app              # optional: filter by metadata.name
  path: spec.replicas
  operator: ==
  expected: 3

check-app-label:
  kind: Deployment
  name: my-app
  path: spec.template.metadata.labels.app
  operator: ==
  expected: my-app

check-image-repo:
  kind: Deployment
  name: my-app
  path: spec.template.spec.containers[0].image
  operator: contains
  expected: my-repo

check-secret-exists:
  kind: Secret
  name: my-secret
  path: data.password
  operator: exists
```

**Merging example** — override a single assertion from a base file:
```yaml
# base-asserts.yaml
check-replicas:
  kind: Deployment
  name: my-app
  path: spec.replicas
  operator: ==
  expected: 1

# prod-asserts.yaml (overrides check-replicas from base)
check-replicas:
  kind: Deployment
  name: my-app
  path: spec.replicas
  operator: ==
  expected: 5
```
```sh
ya assert --assert-file base-asserts.yaml --assert-file prod-asserts.yaml
```

### Operators

| Operator   | Description                                      |
|------------|--------------------------------------------------|
| `==`       | Equal                                            |
| `!=`       | Not equal                                        |
| `contains` | String contains substring, or array contains item |
| `exists`   | Path exists in the document (no value needed)    |

### Path syntax

Paths use dot notation with bracket indexing:

- `spec.replicas`
- `spec.template.spec.containers[0].image`
- `metadata.labels.app`

## Examples

**Merge values files:**
```sh
ya merge base.yaml override.yaml
```

**Assert on Helm output:**
```sh
helm template myrelease ./mychart -f values.yaml | ya assert \
  --assert spec.replicas==3 \
  --assert metadata.labels.app==myrelease
```

**Assert on multiple resources using a file:**
```sh
helm template myrelease ./mychart -f values.yaml \
  | ya assert --assert-file asserts.yaml
```

**Assert on a specific resource kind and name:**
```yaml
# asserts.yaml
check-pull-policy:
  kind: Deployment
  name: myrelease
  path: spec.template.spec.containers[0].imagePullPolicy
  operator: ==
  expected: Always

check-token-exists:
  kind: Secret
  name: myrelease-secret
  path: data.token
  operator: exists
```

---

Contributions and issues welcome!
