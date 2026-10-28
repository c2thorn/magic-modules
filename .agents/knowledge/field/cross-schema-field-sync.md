---
name: cross-schema-field-sync
description: Keep paired *Version.yaml snapshot schemas and TGC is_missing_in_cai flags synchronized when adding fields.
topics: [field, tgc]
task_types: [field-add, new-resource]
source: docs/content/develop/add-fields.md + docs/content/convert/add-new-resource-tgc.md
status: draft
last_verified: 2026-10-28
---

# Cross-Schema Field Sync (`*Version.yaml` & `is_missing_in_cai`)

When adding fields to an MMv1 resource, check sibling schemas and converter flags in the same product directory before generating.

## 1. Paired `*Version.yaml` Snapshot Schemas

Several products expose both a mutable parent resource (for example `mmv1/products/ces/App.yaml` or `mmv1/products/dialogflowcx/Tool.yaml`) and a paired immutable snapshot resource (`AppVersion.yaml`, `ToolVersion.yaml`) whose `snapshot` NestedObject embeds a read-only copy of the parent resource's configuration.

- **Rule:** Whenever you add a field to a resource in `mmv1/products/<product>/<Resource>.yaml`, check for a paired `<Resource>Version.yaml` (or sibling snapshot schema) in the same directory. If the API's version snapshot message exposes the new field, mirror it in the snapshot schema with `output: true` (and nested subfields marked `output: true`).
- **Why:** If the parent resource sets the new field and a test or user creates a `<Resource>Version` snapshot, omitting the field from the snapshot schema drops snapshot state or causes incomplete version reads.

```yaml
# In <Resource>Version.yaml under snapshot.properties:
- name: 'clientCertificateSettings'
  type: NestedObject
  output: true
  description: Settings for custom client certificates captured in this version.
  properties:
    - name: 'tlsCertificate'
      type: String
      output: true
      description: TLS certificate PEM.
```

## 2. `is_missing_in_cai: true` on `include_in_tgc_next: true` Resources

- **Rule:** Check the top of `Resource.yaml` for `include_in_tgc_next: true`. When enabled, Magic Modules also generates `terraform-google-conversion` (TGC) converters mapping Terraform fields onto Cloud Asset Inventory (CAI) proto schemas. If a newly added field does not yet exist in the upstream CAI asset schema for that resource, set `is_missing_in_cai: true` on the field.
- **Why:** Without `is_missing_in_cai: true`, TGC converter generation emits expanders targeting a nonexistent CAI field or causes converter drift/build failures downstream.

```yaml
- name: 'experimentalConfig'
  type: String
  is_missing_in_cai: true
  description: Configuration not yet surfaced in Cloud Asset Inventory.
```

## Do NOT Use For

- Do **not** set `is_missing_in_cai: true` on resources without `include_in_tgc_next: true`, or on fields that already exist in the CAI asset schema.
- Do **not** copy write-only credentials or request-only parameters (`url_param_only: true`) into `*Version.yaml` snapshot schemas if the API omits them from snapshots.
