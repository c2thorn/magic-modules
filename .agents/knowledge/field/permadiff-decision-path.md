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
   - **Caution:** Cannot be combined with `default_value`, and makes clearing back to unset impossible unless paired with an explicit reset value.
3. **API normalizes formatting (case, trailing slash, self-link vs. short name, duration/JSON equivalence)** -> **`diff_suppress_func` or `custom_flatten`**
   - Prefer reusable helpers in `tpgresource` (`CaseDiffSuppress`, `CompareSelfLinkOrResourceName`, `DurationDiffSuppress`, `EmptyOrDefaultStringSuppress`) before writing a custom suppressor.
   - Use `custom_flatten` when the API returns an alternate wire representation that can be canonically normalized on read so `ImportStateVerify` also sees clean state.
4. **Secret, password, or one-way token never returned by `GET`** -> **`write_only: true`** (preferred for new credentials) or **`ignore_read: true`** + `ImportStateVerifyIgnore`
   - Always add a comment beside `ignore_read: true` and `ImportStateVerifyIgnore` explaining the API behavior that omits or redacts the field on `GET`.
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
- **Never suppress diffs unconditionally** (`return true`); always compare normalized `old` and `new` values so genuine user edits still trigger updates.
