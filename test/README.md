# Integration fixtures

This directory contains robot configurations and extension fixtures used by
the process-backed YAML integration suites in `integration/suites/data/`.

Each suite selects a fixture with its `config_dir` field. The integration
runner copies that fixture into an isolated run directory before starting a
real robot process, so generated workspace and brain data do not modify the
source fixture.

Build the runner and list suites with:

```shell
make integration-build
./gopherbot-integration list-suites
```

Run a focused suite with:

```shell
./gopherbot-integration run-suite TestPrompting
```

Run artifacts are stored under `integration/runs/<run-id>/`. See
`aidocs/TESTING_CURRENT.md` for the authoritative validation workflow and
`docs/src/botdev/IntegrationTests.md` for suite-authoring guidance.

The `test` Go module and `util.go` also provide formatting support used by the
terminal connector. They are shared infrastructure, not a standalone Go test
harness.
