# coverage_filter

Small CLI used by `scripts/coverage/test_coverage.sh` to post-process function-coverage violations:

- drop **gqlgen** output under `graph/**/*.go` (when enabled)
- drop **thin delegate** wrappers (single `return` forwarding a direct call with identifier-only args)

Run from repository root:

```bash
go run ./scripts/coverage/coverage_filter --violations /path/to/violations.tsv --repo-root .
```

