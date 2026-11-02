---
name: zero-value-clearing
description: Clear mutable fields back to false, 0, "", or {} on PATCH using send_empty_value, allow_empty_object, or GetRawConfig.
topics: [field, custom-code]
task_types: [field-add, new-resource, bug-fix]
source: docs/content/reference/field.md + mmv1 template behavior
status: draft
last_verified: 2026-11-02
---

# Zero-Value Clearing (`false`, `0`, `""`, `{}`) & Raw Config Access

## 1. `send_empty_value: true` on Mutable Primitive Fields

By default, generated MMv1 expanders check `tpgresource.IsEmptyValue(reflect.ValueOf(...))` and **omit** Go zero values (`false`, `0`, `""`, empty slices) from request JSON bodies.

- **Why this causes update bugs:** During `PATCH` updates, MMv1 adds changed fields to `?updateMask=...` via `d.HasChange(...)`. If a user updates a mutable field from a non-zero value back to its zero value (for example `true -> false` or `"foo" -> ""`), `updateMask` lists the field while the request body omits it—causing the API to reject the request, ignore the reset, or fail to clear the value, resulting in a post-apply permadiff (`expected false, got true`).
- **Rule:** Set `send_empty_value: true` on mutable `Boolean` (and mutable `String` / `Integer` / `Double`) fields whenever setting `false`, `""`, or `0` is a valid update state or resets a previously set non-zero value.

```yaml
- name: 'includeExecution'
  type: Boolean
  description: |
    Whether execution details should be included in the response.
  send_empty_value: true
```

Always include a two-step acceptance test transition from non-zero to zero (for example `true` in Step 1 -> `false` in Step 2) to verify the `PATCH` payload actually clears the field on the server.

## 2. `allow_empty_object: true` on Presence-Based Nested Blocks

- **Rule:** When an API represents feature activation as an empty JSON object (`"featureConfig": {}`), set `allow_empty_object: true` (and `send_empty_value: true` when an explicit `{}` must be serialized on create/update) on the `NestedObject`.
- **Why:** Without `allow_empty_object: true`, MMv1 treats an all-empty block as empty and strips `{}` from the outgoing payload.

## 3. Distinguishing Unset (`null`) vs. Zero Value in Custom Go Code

`d.Get("field")` and `d.GetOk("field")` cannot distinguish between a field omitted from HCL (`null`) and a field explicitly configured to its Go zero value (`false`, `0`, `""`).

### Top-Level Fields vs. Handwritten / Custom Diff Code

- **Rule:** In custom expanders, encoders, `custom_diff`, or handwritten CRUD code where "omitted by user" and "explicitly set to `false` / `0` / `""`" have different API semantics, inspect `d.GetRawConfig()` (or `diff.GetRawConfig()` inside `CustomizeDiff`):

```go
rawCfg := d.GetRawConfig()
if !rawCfg.IsNull() && rawCfg.IsKnown() {
	val := rawCfg.GetAttr("enabled")
	if !val.IsNull() && val.IsKnown() {
		req["enabled"] = val.True()
	}
}
```

### Nested Fields with `default_from_api` (Custom Expand with Raw Config Inspection)

When a nested field inside an object needs explicit zero values (`false` or `0`) sent on the wire while leaving unconfigured fields omitted so the API default applies, implement a `custom_expand` template using `rd.GetRawConfigAt(path)` and returning a pointer (`*bool` or `*int64`, or `nil` when unset), guarded for non-TGC compilers.

Returning a pointer allows the expander to distinguish an explicit zero from an unconfigured field: a non-nil pointer (`&f` or `&zero`) is recognized as non-empty and serialized in the request body, while `nil` omits the field entirely so the API default is preserved.

```go
func expand{{$.GetPrefix}}{{$.TitlelizeProperty}}(v interface{}, d tpgresource.TerraformResourceData, config *transport_tpg.Config) (interface{}, error) {
	b, ok := v.(bool)
	if !ok {
		return nil, nil
	}
	if b {
		return b, nil
	}
{{- if not (contains $.ResourceMetadata.ProductMetadata.Compiler "terraformgoogleconversion") }}

	if rd, ok := d.(*schema.ResourceData); ok {
		path := cty.GetAttrPath("parent_block").IndexInt(0).GetAttr("{{ underscore $.Name }}")
		if val, _ := rd.GetRawConfigAt(path); !val.IsNull() && val.IsKnown() {
			f := false
			return &f, nil
		}
	}
{{- end }}

	return nil, nil
}
```

#### Live Reference Templates

Refer to live implementations in the repository:
- `mmv1/templates/terraform/custom_expand/compute_backend_service_cdn_policy_ttl.go.tmpl` (integer TTL returning `*int64` or `nil`)
- `mmv1/templates/terraform/custom_expand/modelarmor_template_data_residency_compliant.go.tmpl` (boolean returning `*bool` or `nil`)

## Do NOT Use `send_empty_value: true` For

- **Create-only / `immutable: true` fields** where omitting the zero value on `POST` already matches the API's default behavior and no `PATCH` transition is possible.
- **Optional fields whose zero value is rejected by the API** when unset (for example an optional string field where sending `""` triggers an API validation error instead of omitting the key).
- **`output: true` (computed-only) fields**, which are never sent in request bodies.
