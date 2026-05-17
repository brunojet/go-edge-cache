# Copilot Instructions

## `internal` and `pkg` Structure

- Keep all internal application logic in `internal/`; `pkg/` should contain only public contracts, types required by those contracts, and small exposure facades.
- Prefer organizing implementations as `internal/<feature>/<implementation>/`, for example `internal/handlers/net_http/`, and use responsibility-based file names such as `net_http_handler.go`, `net_http_type.go`, `net_http_helper.go`, or `net_http_local.go`.
- In `pkg/<feature>/`, expose only the public surface: contracts under `contracts/` when useful to avoid import cycles, and `api.go` as the central point for re-exports, aliases, wrappers, and public entrypoints.
- Keep mocks in `pkg/<feature>/mocks/` as a sibling of `contracts/` so public contracts and test doubles remain discoverable and consistently organized.
- If code in `pkg/` starts accumulating business logic, orchestration rules, or infrastructure details, move the implementation to `internal/` and keep only the public API in `pkg/`.

## Documentation in `pkg`

- `pkg/` is the module public surface; everything under this path must be documented for GoDoc generation (static or server mode).
- Every package in `pkg/` must have a package comment, and every exported symbol (types, interfaces, functions, methods, constants, and variables) must have a clear and objective GoDoc comment.
- When creating or changing code in `pkg/`, update documentation in the same PR/task; do not leave exported symbols undocumented.
- Prefer contract-and-usage-focused comments: purpose, semantics, relevant invariants, and short usage examples when helpful.

## Test File Mapping

- For every `*.go` file created or modified, keep a matching `*_test.go` file in the same package and update it in the same task.
- Each `*_test.go` file must cover only the functions, methods, helpers, and behaviors owned by its matching source file. Do not use one test file to absorb coverage for unrelated production files unless the code is inseparable and that limitation is stated explicitly in the conversation.
- If a file exists only for types, contracts, mocks, or re-exports, still verify whether a matching `*_test.go` should exist; if there is no meaningful behavior to test, state that explicitly in the conversation before ending the task.

## Unit Test Standards

- Keep tests deterministic, isolated, and fast. Avoid network access, real external services, sleeping, shared mutable global state, and filesystem side effects outside test temp directories unless the test is explicitly validating that behavior.
- Prefer file-scoped tests with narrow responsibility boundaries so coverage can be reached with fewer branches per test file and with clearer ownership of failures.
- Prefer table-driven tests when they reduce duplication without hiding intent, and use small focused assertions with clear failure messages.
- Cover success paths, owned error paths, and relevant boundaries. Use mocks, fakes, or stubs only at dependency boundaries, keep fixtures minimal, and prefer explicit setup or small local helpers over shared frameworks.