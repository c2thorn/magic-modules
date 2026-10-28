---
name: enums-vs-strings
description: Model an API enum as Enum (strict, plan-time) or String (forward-compatible) - the deliberate trade-off.
topics: [field]
task_types: [field-add, new-resource]
source: docs/content/best-practices/validation.md + the schema-review skill's original trade-off note
status: draft
last_verified: 2026-10-28
---

# Enum vs. String

**The trade-off (decide deliberately, per field):**
- **`Enum`** when the value set is exhaustive for a clearly-defined domain and new values are extremely
  unlikely. Buys strict plan-time validation — users fail fast instead of mid-apply.
- **`String`** (with allowed values named in the description and a link to API docs) when the API is
  likely to add values. **Why:** an Enum's value list lives in the provider; when the API adds a value,
  every user needs a provider upgrade before they can use it — and imports of resources using the new
  value break. A String stays forward-compatible.

```yaml
- name: 'severity'
  type: Enum
  enum_values:
    - 'LOW'
    - 'MEDIUM'
    - 'HIGH'
```

**Rules for Enum (MMv1 YAML):**
- Omit the `FIELD_NAME_UNSPECIFIED` value from `enum_values` — it is the proto convention's first enum
  value, representing 0/unset, not a real choice.
- Do **not** manually write `Possible values:` or `Possible values are:` in `description:` on a `type: Enum`
  field (or an `Array` of `Enum`). **Why:** MMv1 templates (`property_documentation.html.markdown.tmpl` and
  `schema_property.go.tmpl`) automatically append `Possible values are: ...` / `Possible values: [...]`
  from `enum_values`, so repeating them in `description:` duplicates the list in generated website docs.
- Enums validate against the list automatically; a custom `validation` **overrides** that default — if you
  add one, it must re-verify the enum values itself.

**Rule for String-instead-of-enum (MMv1 YAML):** list the allowed values in the field `description:` with a
link to the API docs (since MMv1 does not auto-append `Possible values` for `type: String`) — otherwise
users only discover an invalid value when the API rejects it mid-apply.

**Rule for Handwritten Go Schemas (`validation.StringInSlice`):**
- On `Optional: true, Computed: true` `TypeString` fields without diff suppression, omit proto `0`-value
  `*_UNSPECIFIED` entries from `validation.StringInSlice` unless the API genuinely persists and distinguishes
  `*_UNSPECIFIED` as a user-settable or reset value.
- **Why:** Allowing users to configure `"FOO_UNSPECIFIED"` when the API omits or normalizes unset values on
  `GET` (or returns `"FOO_UNSPECIFIED"` when omitted) creates immediate unset-vs-`*_UNSPECIFIED` permadiffs.
