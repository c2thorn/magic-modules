---
name: import-safe-flatteners
description: Write import-safe flatteners and handle server-populated defaults on ForceNew fields without breaking terraform import.
topics: [field, custom-code, test]
task_types: [field-add, new-resource, bug-fix, code-review]
source: docs/content/develop/custom-code.md + docs/content/develop/diffs.md + mmv1 template behavior
status: approved
last_verified: 2026-11-02
---

# Import-Safe Flatteners, `ForceNew` Server Defaults, & Write-Only Parent Blocks

## 1. Unconditional API Population for Readable Fields

Flatteners for fields returned by the API on `GET` must populate values unconditionally from the API response `v` (or response map `res`).

- **Why `d.Get(...)` in flatteners breaks import:** During `terraform import`, `ResourceData` `d` contains only `id`. If a flattener checks `d.Get("some_field")` to decide whether to populate state, `d.Get` returns an empty value and the flattener omits or zeros out the field, failing to capture full server state on import.
- **The only valid exception:** Reserve reading prior state via `d.Get(...)` inside a flattener strictly for genuine write-only or sensitive inputs that the API intentionally omits on `GET` (such as `initial_user.0.password` in `mmv1/templates/terraform/custom_flatten/alloydb_cluster_input_user_flatten.go.tmpl`). Always pair this exception with `ImportStateVerifyIgnore` in acceptance tests.

```go
// Correct flattener for readable API fields: populate from API response value `v`
func flattenProductResourceField(v interface{}, d *schema.ResourceData, config *transport_tpg.Config) interface{} {
	if v == nil {
		return nil
	}
	return v
}
```

## 2. Server-Populated Defaults on `ForceNew` Fields (`d.GetOk` Trap)

When an unconfigured `ForceNew` field receives a default value populated by the server on `GET`, handle the diff using a `DiffSuppressFunc` or a separate computed `effective_*` attribute rather than branching on `d.GetOk` inside `Read`.

- **The Trap:** Attempting to prevent a diff by adding `if _, ok := d.GetOk("field"); !ok { return nil }` in `Read` or in a flattener causes post-import resource recreation bugs:
  1. On initial `apply` with an unconfigured field, `d.GetOk` is false, so state remains empty and no diff is shown.
  2. On `terraform import`, the provider populates server values directly into prior state.
  3. On the next `terraform plan`, `d.GetOk("field")` now finds the imported server value in state and compares it against the user's empty configuration. Because the field is `ForceNew`, Terraform plans a destructive resource recreation.
- **Rule:** Keep `Read` and flatteners faithful to server state. Suppress plan diffs using a `DiffSuppressFunc` (such as `tpgresource.EmptyOrDefaultStringSuppress(...)` in `mmv1/products/compute/Address.yaml`) or expose the server-defaulted value via a separate `effective_*` attribute.

## 3. Parent `NestedObject` Blocks with All Write-Only / `ignore_read` Children

When every child property inside an MMv1 `NestedObject` uses `ignore_read: true` or `write_only: true` and the API omits the parent block on `GET`:

- **Rule:** Set `ignore_read: true` on the parent `NestedObject` as well so state readback matches the omitted response.
- **Live Reference:** See `mmv1/products/agentidentity/AuthProvider.yaml` lines 78–91, where the parent `apiKey` `NestedObject` and its child `apiKey` string both set `ignore_read: true`.
