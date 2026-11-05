---
name: permadiff-decision-path
description: Choose the right schema attribute or custom hook to eliminate post-apply diffs without masking real drift.
topics: [field, custom-code]
task_types: [field-add, new-resource, bug-fix, test-fix]
source: docs/content/develop/diffs.md + docs/content/reference/field.md
status: draft
last_verified: 2026-10-28
---

# Permadiff Decision Path

When `terraform plan` is non-empty immediately after `apply` (or during `ImportStateVerify`), match the root cause to the narrowest fix rather than reaching for `ignore_read: true`.

## 1. Choose the Right Mechanism

1. **Static, predictable API default** -> **`default_value: <val>`**
   - **Why:** Gives users plan-time visibility of the exact default without marking the field `Computed`.
2. **Dynamic, server-derived, or environment-dependent default** -> **`default_from_api: true`** (`Optional` + `Computed`)
   - **Why:** Tells Terraform to adopt the API-populated value into state when omitted in HCL while still enforcing planned changes when configured.
   - **Rule:** Set `default_from_api: true` only when omitting the field on `Create` causes the API's `GET` response to return a materialized non-empty value on the wire. When an API description mentions a conceptual default while `GET` omits the field when unset, keep the field pure `Optional` so users retain the ability to unset or clear it in HCL.
   - **Caution:** Cannot be combined with `default_value`, and makes clearing back to unset impossible unless paired with an explicit reset value.
3. **API normalizes formatting (case, trailing slash, self-link vs. short name, duration/JSON equivalence)** -> **`diff_suppress_func` or `custom_flatten`**
   - Prefer reusable helpers in `tpgresource` (`CaseDiffSuppress`, `CompareSelfLinkOrResourceName`, `DurationDiffSuppress`, `EmptyOrDefaultStringSuppress`) before writing a custom suppressor.
   - Use `custom_flatten` when the API returns an alternate wire representation that can be canonically normalized on read so `ImportStateVerify` also sees clean state.
4. **Secret, password, or one-way token never returned by `GET`** -> **`write_only: true`** (preferred for new credentials) or **`ignore_read: true`** + `ImportStateVerifyIgnore`
   - Use `ignore_read: true` (or `write_only: true`) strictly for sensitive one-time inputs (such as OAuth authorization codes, initial setup tokens, or write-only passwords) accepted on `POST`/`PATCH` and intentionally omitted by the API on `GET`. Normal fields returned by the API on `GET` must be flattened directly from the response rather than using `ignore_read: true`.
   - Always add a comment beside `ignore_read: true` and `ImportStateVerifyIgnore` explaining the API behavior that omits or redacts the field on `GET`.
   - When every child inside a `NestedObject` uses `ignore_read: true` or `write_only: true` and the API omits the parent block on `GET`, set `ignore_read: true` on the parent `NestedObject` as well (see `mmv1/products/agentidentity/AuthProvider.yaml`).
5. **Action/trigger inputs or mutually-exclusive update transitions** -> **`custom_code.update_encoder`, `custom_diff`, or targeted `DiffSuppressFunc`**
   - **Why:** Some fields act as update-time triggers, state-transition selectors, or mutually-exclusive `oneof` inputs where `update_encoder` strips stale sibling fields or `custom_diff` / `DiffSuppressFunc` reconciles the post-transition state. These are valid, idiomatic patterns—do **not** replace working encoder/diff hooks with `ignore_read: true` unless the API genuinely never returns the field on `GET`.

```yaml
- name: 'etag'
  type: String
  description: Server-computed checksum for optimistic concurrency.
  output: true
```

## 2. Do NOT Use For

- **Never use `ignore_read: true` or `output: true` to silence a broken expander, missing `updateMask` entry, or zero-value clearing bug** (see [zero-value-clearing.md](zero-value-clearing.md)).
  - `ignore_read: true` blinds Terraform to out-of-band drift and requires `ImportStateVerifyIgnore`.
  - `output: true` makes the field read-only and removes user configurability completely.
- **Never branch on `d.GetOk` inside `Read` or flatteners on `ForceNew` fields** to avoid diffs from server-populated defaults. `terraform import` writes the server value into state; a `d.GetOk` check on subsequent plans finds the imported value and forces resource recreation. Use `DiffSuppressFunc` (such as `tpgresource.EmptyOrDefaultStringSuppress(...)`) or a separate computed `effective_*` attribute instead (see [import-safe-flatteners.md](import-safe-flatteners.md)).
- **Never suppress diffs unconditionally** (`return true`); always compare normalized `old` and `new` values so genuine user edits still trigger updates.
