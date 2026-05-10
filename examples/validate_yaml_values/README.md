# Validate YAML Value Files Use Case

## Overview

This example shows how to validate and test Helm values files before rendering charts. By merging multiple values files and running assertions against the computed configuration, you can catch misconfigurations, missing keys, or incorrect overrides early—before they reach your cluster. This workflow is ideal for CI pipelines or local development to ensure your values files produce the intended result.


## Command - Merge values files to see final result

```sh
./ya merge \
./testdata/base.yaml \
./testdata/override_1.yaml \
./testdata/override_2.yaml
```

## Command - Test merged values files (Happy Path)

The following command can be run to merge values files and run assertions on the computed result:

```sh
./ya merge \
./testdata/base.yaml \
./testdata/override_1.yaml \
./testdata/override_2.yaml | \
./ya assert \
--assert secret.password=="test-pass"
```

This should return an output similar to the following, which is an indicator of success:

```
PASS: secret.password==test-pass [/] secret.password == test-pass
```

## Command - Test merged values files (Unhappy Path)

The following command can be run to merge values files and run assertions on the computed result, although this one is an example of what failure looks like:

```sh
./ya merge \
./testdata/base.yaml \
./testdata/override_1.yaml \
./testdata/override_2.yaml | \
./ya assert \
--assert secret.password=="test-pass1123123"
```

This should return an output similar to the following, which shows indicators of failure:

```
FAIL: secret.password==test-pass1123123 [/] secret.password == test-pass1123123
```