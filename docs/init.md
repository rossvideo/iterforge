# Scaffolding a Project

`iterforge init` generates a self-contained, working IterForge project from an
embedded workflow template.

```bash
go run ./cmd/iterforge init <project-name>                     # default template
go run ./cmd/iterforge init -template extraction <name>        # pick a template
go run ./cmd/iterforge init -dir <parent> -template <t> <name> # choose parent dir
```

It refuses to write into a non-empty destination and errors on an unknown
template. Run `go run ./cmd/iterforge init -h` to see the available templates.
See [workflow-templates.md](workflow-templates.md) for the catalog.

## Generated layout

```
<name>/
├── README.md
├── ITERFORGE.md
├── Makefile
├── policy.yaml
├── go.mod                 # module <name>
├── candidates/            # mutable surface
│   ├── candidate.go
│   └── candidate_test.go
├── evals/                 # frozen evaluator
│   ├── evaluator.go
│   ├── evaluator_test.go
│   └── golden_set.jsonl
├── cmd/
│   ├── runexp/main.go     # self-contained scored runner
│   └── summarize/main.go
└── logs/
    ├── agent_journal.md
    └── results.jsonl      # empty, append-only
```

The generated project is self-contained — it does not depend on this repo's
internal packages — and passes out of the box:

```bash
cd <name>
make check
make baseline
make summarize
```

## How templates work

Template files live under `internal/templates/files/` in two kinds of layer:

- `_shared/` — files common to every template (`go.mod`, `Makefile`, `cmd/`,
  `logs/`);
- `<template>/` — files specific to one workflow (candidate, evaluator, golden
  set, `policy.yaml`, docs).

`Init` writes the `_shared` layer, then the chosen template layer on top. All
files carry a `.tmpl` suffix so the parent module's `build`/`vet`/`test`/`gofmt`
never treat them as source; on generation the suffix is stripped and the
`__MODULE__` placeholder is replaced with the project name (the Go module path).

To add a template, create `files/<name>/` with at least `candidates/`, `evals/`,
`policy.yaml.tmpl`, `README.md.tmpl`, `ITERFORGE.md.tmpl`. Keep the `.tmpl` Go
files gofmt-canonical (generate once, run `gofmt -l` on the output to verify).
