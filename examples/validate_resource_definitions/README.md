## Validate Helm YAML Resource Definitions Use Case

## Overview

This example demonstrates how to use ya as a Helm post-renderer to validate rendered Kubernetes resource definitions. By specifying assertions in YAML files, you can enforce structural and content requirements on your manifests to ensure that critical fields, labels, environment variables, and secrets are present and correctly configured before resources are applied. This approach enables automated, testable policy enforcement in CI/CD pipelines or local development. This can be run as part of helm `template`, `upgrade` or `install` operations, and will prevent deployment if all assertions have not passed.

## Command - Happy Path

Running the following command within this repository for this use case after compiling the binary should look like the below. This should result in the chart being renderered as normal successfully. This is a sign that the assertions have passed.

```sh
helm template testing ./testdata/testchart \
-f ./testdata/base.yaml \
-f ./testdata/override_1.yaml \
-f ./testdata/override_2.yaml \
--post-renderer ./ya \
--post-renderer-args post-render \
--post-renderer-args --assert-file \
--post-renderer-args ./testdata/assert/deployment_assert.yaml \
--post-renderer-args --assert-file \
--post-renderer-args ./testdata/assert/secret_assert.yaml
```

## Command - Unhappy Path

Running the following command within this repository for this use case after compiling the binary should look like the below. This should result in the chart returning that an error occured, and showing the output of each of the tests, which should allow the user to clearly see what failed. This is a sign that the assertions have failed.

```sh
helm template testing ./testdata/testchart \
-f ./testdata/base.yaml \
-f ./testdata/override_1.yaml \
-f ./testdata/override_2.yaml \
--post-renderer ./ya \
--post-renderer-args post-render \
--post-renderer-args --assert-file \
--post-renderer-args ./testdata/assert/deployment_assert.yaml \
--post-renderer-args --assert-file \
--post-renderer-args ./testdata/assert/secret_assert.yaml
--post-renderer-args --assert-file \
--post-renderer-args ./testdata/assert/secret_fail_assert.yaml
```

Output should render to be something like the below:

```
Error: error while running post render on files: error while running command <path> error output:
PASS: check-app-label [Deployment/test-value-override] spec.template.metadata.labels.app == test-value-override
PASS: check-env-var [Deployment/test-value-override] spec.template.spec.containers[0].env contains map[name:ENV_VAR]
PASS: check-image-contains-repo [Deployment/test-value-override] spec.template.spec.containers[0].image contains test-repo
PASS: check-image-exists [Deployment/test-value-override] spec.template.spec.containers[0].image exists <nil>
PASS: check-image-pull-policy [Deployment/test-value-override] spec.template.spec.containers[0].imagePullPolicy == Always
PASS: check-secret-password [Secret/test-secret] data.password exists <nil>
PASS: check-secret-username [Secret/test-secret] data.username exists <nil>
FAIL: fail-check-user-id [Secret/test-secret] data.user-id exists <nil>
: exit status 1
```