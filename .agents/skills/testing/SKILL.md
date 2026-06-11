---
name: unit-and-integration-testing
description: Proactively use when writing, updating, fixing, or reviewing tests (unit/integration), adding test coverage, creating mocks/fixtures, or changing app code that requires test updates.
---

# Go testing

## When to use this skill

- MUST use for any request mentioning: test, tests, testing, test planning, unit test, integration test, coverage, mock, fixture.
- MUST use when app code is added/changed/refactored and tests need to be created or updated.
- MUST use when fixing bugs via tests first (repro test) or updating broken/flaky tests.
- Use proactively even if user asks only for feature/code changes and tests are implied by repo standards.

## General rules

- One test file per one original code file. Do not put test cases for separate original files into single test file.
- Maintain unit tests order based on order of methods in original file. Happy path / success test cases should go first, edge cases later.
- Use `t.Parallel()` where appropriate.

## Coverage

- Aim to have a broad coverage, preferably to 100% unit + integration tests combined.
- Try to provide unit tests coverage for all code paths except for those that require external resources. Writing tests for business logic with external dependencies mocked is ok. Repository tests may use in-memory SQLite when validating SQL behavior directly is valuable.

## Asertions

- NEVER create helpers like "run test case body". All assertions should happen inside loop body itself. Prefer shallow tests (keep minimal number of successive assertions that require previous assertions to hold)

## Test naming

- Test names and comments must describe the guaranteed business behavior and observable consequence, not just the code branch or dependency that returns an error.
- Avoid implementation terms in test titles; name the user-visible consequence instead, especially for errors: what is not created, changed, deleted, returned, or shown.
- Add a comment in English before each top-level test function. It should describe the test case for a single test, or the test group if the test function contains table tests or multiple t.Run() calls.
- Each test case and subcase should have a clear title in English. This applies just to titles, keep communicating with user using language they started the dialog. We treat tests as documentation, so glancing at tests should give developer a complete understanding what this code does and why each branch of logic exists.
- Test names must describe behavior that is explicitly asserted in the test; do not name unasserted side effects or skipped calls unless the test verifies them directly.
- Test names must describe only behavior explicitly asserted in the test: returned error/value, persisted state, produced status, or directly verified mock call. Do not name skipped calls or side effects unless the test verifies them directly.
- Avoid generic titles like "returns error ..."; prefer the business consequence, e.g. "does not sync completed TickTick task twice" or "reports token refresh failure without overwriting stored tokens".
- If any helpers are created for mock setup, add a comment in English before such helper describing which exact case it sets up.

## Table tests

- Setup shared values for test cases in outer test body. Do not repeat same table value multiple times for multiple tests.
- Avoid Unnecessary Complexity in Table Tests. Table tests should NOT be used whenever there needs to be complex or conditional logic inside subtests (i.e. complex logic inside the for loop).
- Do not use table tests just to avoid writing two or three explicit subtests. If cases have different setup, different assertions, or different business meaning, prefer separate `t.Run` blocks.
- Large, complex table tests should be split into either multiple test tables or multiple individual Test... functions.
- Do not use large functions in table tests setup. If function that provides table value takes more than 3 lines, that is separate test case setup that should be executed via t.Run or just a separate test case.

## Mocks

- Use mockgen. All mocks should go into `mocks` subpackage, file named mock.go for conventional interfaces.go original file.
- If multiple files in package defines interfaces, it is ok to generate separate mock in `mocks` package for each of them.

```
//go:generate go tool mockgen -source=interfaces.go -destination=mocks/mock.go -package=mocks
```

- Never edit mock files manually. Use `make generate` to refresh mocks.
- Never use manually written stubs/mocks/etc. Always use generated mocks.
- If complex mock setup reused in multiple test cases, extract it to expectMockSetupName helper function. Helper names must be specific and descriptive.
- Mock setup helpers are allowed only when they remove meaningful repetition; tiny helpers that only wrap `gomock.NewController` or one mock constructor should usually be inlined.
- If a mock setup helper is introduced, add an English comment explaining which exact scenario or dependency behavior it sets up.

## Integration tests

- Use a mock server (e.g. `httptest.Server`) to emulate external HTTP services such as the TickTick API. Configure clients to point at the mock server URL in tests.
- Use in-memory SQLite for storage in integration tests, mirroring production driver behavior without external infrastructure.
- Reset database state between tests via migrations or a shared setup helper.
- Use `t.Parallel()` only when each test has isolated in-memory SQLite state and its own mock server or otherwise independent mock server state.
- Keep unit tests for pure logic; use integration tests for flows that exercise repository SQL, HTTP clients, and wiring together.

## Fixtures

- If same set of test data used in two or more tests, it should be extracted to fixtures.
- Use functional fixtures pattern. Maintain default set of data. Override some fixtures on test case level if data changes are required. Prefer using existing fixture. Extend them as needed.

## Tooling

- use `make test` to run whole test suite
