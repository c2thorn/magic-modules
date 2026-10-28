---
name: unit-test-scope
description: When to write unit tests vs acceptance tests, and avoiding test bloat.
topics: [test]
task_types: [bug-fix, test-fix, field-add, new-resource]
source: authored
status: draft
last_verified: 2026-10-28
---

# Unit Test Scope & Bloat Prevention

Unit tests in `mmv1/third_party/terraform/services/*/*_test.go` and `*_internal_test.go` verify isolated Go logic. They should target reusable logic and systemic patterns, not one-off declarative schema definitions.

## When to Write Unit Tests

Write standalone Go unit tests for custom functions with non-trivial algorithmic logic, branching, normalization, or parsing:
- **Complex diff suppress functions** (`*DiffSuppress`): functions with custom normalization, parsing, or multi-case equivalence rules (not reuse of existing `tpgresource.*` suppressors or trivial single-comparison wrappers).
- **Custom validation functions** (`ValidateFunc`) & **compiled regexes**: testing edge-case inputs, boundary conditions, or domain-specific input rules.
- **State upgraders & ID/URL parsers**: string parsing, URL normalization, or resource ID deconstruction.
- **Complex custom expanders/flatteners**: non-trivial data transformations with branching or edge cases that are hard to cover exhaustively in live acceptance tests.

**Why:** These functions contain custom execution logic with edge cases that could regress silently if modified. For a reference implementation, see [Add unit tests (`docs/content/test/test.md`)](../../../docs/content/test/test.md#add-unit-tests).

## Do NOT Use Unit Tests For

- **Built-in Helpers & Standard CRUD Hooks**: Do NOT require standalone unit tests when wiring an existing `tpgresource.*` diff suppressor/validator, or for straightforward `encoder`/`decoder`/`update_encoder` hooks whose behavior is directly exercised by `TestAcc*` acceptance tests.
- **Declarative Schema Properties**: Do NOT write unit tests that mock `r.Validate(c)` simply to check declarative schema properties (e.g. `Required`, `Optional`, `AtLeastOneOf`, `ConflictsWith`, `Default`).
  - **Why:** The Terraform SDK already tests its schema engine. Testing single declarative field permutations adds high-maintenance test bloat.
- **One-Off Typo Fixes**: Do NOT add persistent unit tests for one-off typos or copy-paste slice errors.
  - **Why:** These are covered by standard acceptance tests (`TestAcc*`) or verified ephemerally during triage reproduction.
- **Ephemeral Debug Tests**: Remove any ad-hoc unit tests created solely to reproduce an issue locally before finalizing a PR.
