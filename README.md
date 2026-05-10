# YAML Assertions (ya)

A CLI tool to merge Helm values files and assert conditions on rendered YAML output, designed for Helm workflows and CI pipelines.

## Features
- Merge multiple YAML values files with Helm-aware coalescing (base + overrides)
- Pipe multi-document YAML (e.g., from `helm template`) directly into `ya`
- Assert values across multiple Kubernetes resources using inline or file-based assertions
- Filter assertions by Kubernetes resource `kind` and `metadata.name`
- Use as a **Helm post-renderer** to gate `helm install`/`upgrade` on assertions passing
- Colored, leveled output (`PASS`/`FAIL`/`ERROR`) with correct exit codes for CI

## Installation

Build from source:
```sh
go build -o ya ./cmd/ya
```

## Usage

`ya` has three subcommands: `merge`, `assert`, and `post-render`.

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

### post-render

`post-render` implements the [Helm post-renderer protocol](https://helm.sh/docs/topics/advanced/#post-rendering). It allows `ya` to gate a `helm install`, `helm upgrade`, or `helm template` on assertions passing — if any assertion fails, Helm aborts with an error.

How it works:
1. Helm renders chart templates and pipes the YAML to `ya`'s stdin.
2. `ya` writes the YAML **unchanged** to stdout (Helm reads this back).
3. All `PASS`/`FAIL`/`ERROR` output goes to **stderr** so it doesn't corrupt the YAML stream on stdout.
4. `ya` exits non-zero if any assertion fails — Helm surfaces this as an error and aborts.

```sh
ya post-render [--assert <expr>] [--assert-file <file>]
```

Accepts the same `--assert` and `--assert-file` flags as the `assert` subcommand. Input always comes from stdin (as required by the Helm protocol) — file arguments are not supported.

#### Helm `--post-renderer-args` convention

Helm passes post-renderer arguments one word per `--post-renderer-args` flag. Each flag value must be a single token — flags and their values are passed separately:

```sh
# Each argument is its own --post-renderer-args flag
helm template myrelease ./mychart \
  --post-renderer ya \
  --post-renderer-args post-render \
  --post-renderer-args --assert-file \
  --post-renderer-args ./asserts.yaml
```

To pass multiple assert files, repeat `--assert-file` and its value as separate pairs:

```sh
helm template myrelease ./mychart \
  --post-renderer ya \
  --post-renderer-args post-render \
  --post-renderer-args --assert-file \
  --post-renderer-args ./base-asserts.yaml \
  --post-renderer-args --assert-file \
  --post-renderer-args ./extra-asserts.yaml
```

To use an inline `--assert`:

```sh
helm template myrelease ./mychart \
  --post-renderer ya \
  --post-renderer-args post-render \
  --post-renderer-args --assert \
  --post-renderer-args "spec.replicas==3"
```

> **Note:** `ya` must be on your `PATH` (or referenced by path) for Helm to invoke it. Build with `go build -o ya ./cmd/ya`.

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

**Gate `helm template` on assertions (useful in CI):**
```sh
helm template myrelease ./mychart -f values.yaml \
  --post-renderer ya \
  --post-renderer-args post-render \
  --post-renderer-args --assert-file \
  --post-renderer-args ./asserts.yaml
```

**Gate `helm install` on assertions:**
```sh
helm install myrelease ./mychart -f values.yaml \
  --post-renderer ya \
  --post-renderer-args post-render \
  --post-renderer-args --assert-file \
  --post-renderer-args ./asserts.yaml
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

## Automated Releases & Versioning

This project uses [release-please](https://github.com/googleapis/release-please) for automated semantic versioning and GitHub Releases.

- **Version bumps and changelogs are generated automatically from commit messages.**
- **Binaries for Linux, macOS, and Windows are published to each GitHub Release.**

### How it works
- All commits after v0.1.0 must follow [Conventional Commits](https://www.conventionalcommits.org/) (e.g., `fix: ...`, `feat: ...`, `feat!: ...`).
- When you push to `main`, release-please will open a PR to bump the version and update the changelog.
- Merging that PR creates a new tag and triggers the build workflow, which publishes binaries to the release.

### Example commit messages
- `fix: correct assertion logic` → patch bump (0.1.0 → 0.1.1)
- `feat: add new operator` → minor bump (0.1.0 → 0.2.0)
- `feat!: breaking change` or `BREAKING CHANGE:` in body → major bump (0.1.0 → 1.0.0)

See the [Releases](https://github.com/ceejaykelly/yaml-assertions/releases) page for downloads.

---

Contributions and issues welcome!
