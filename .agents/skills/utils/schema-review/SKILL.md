---
name: schema-review
description: "Skill for reviewing Magic Modules schemas across Enum/String modeling, zero-value clearing, permadiff handling, and sibling schema sync."
---

# `schema-review`

> **Note to AI Agents:** You MUST read the YAML frontmatter above first. Only read the rest of this file if the `description` matches your current roadblock or required task.

## Prerequisites
* You must be in the `magic-modules` root directory.
* You must be reviewing or adding a field or resource schema.

## Execution Steps

### 1. Enum vs. String & Description Duplication
When adding or reviewing a field backed by an API enum, read [.agents/knowledge/field/enums-vs-strings.md](../../../knowledge/field/enums-vs-strings.md):
* Verify the `Enum` vs. `String` choice, ensure `*_UNSPECIFIED` is omitted (including in handwritten `validation.StringInSlice` on `Optional` + `Computed` fields), and confirm `type: Enum` descriptions do **not** manually duplicate `Possible values:` (which MMv1 generates automatically).

### 2. Zero-Value Clearing, Permadiffs, & Sibling Schemas
Check the relevant field entries in [.agents/knowledge/index.md](../../../knowledge/index.md) when applicable:
* Mutable `Boolean`/primitive fields or empty `{}` blocks: [.agents/knowledge/field/zero-value-clearing.md](../../../knowledge/field/zero-value-clearing.md)
* Defaults, diff suppression, or update hooks: [.agents/knowledge/field/permadiff-decision-path.md](../../../knowledge/field/permadiff-decision-path.md)
* Paired `*Version.yaml` snapshots or `include_in_tgc_next: true` resources: [.agents/knowledge/field/cross-schema-field-sync.md](../../../knowledge/field/cross-schema-field-sync.md)

> [!NOTE]
> The judgment content lives in [.agents/knowledge/](../../../knowledge/index.md) — this skill references those entries and does not hold its own copy.

### 3. Verification & Handoff

* Proceed with your field addition or PR review based on the checklist findings.
