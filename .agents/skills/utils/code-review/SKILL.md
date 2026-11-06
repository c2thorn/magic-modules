---
name: code-review
description: "Review a Magic Modules pull request, branch, or diff for API contract alignment, state lifecycle safety, backward compatibility, and test adequacy. Default skill for single-pass code reviews."
---

# `code-review`

> **Note to AI Agents:** You MUST read the YAML frontmatter above first. Only read the rest of this file if the `description` matches your required task.

Read **both** [`docs/content/code-review/review-pr.md`](../../../../docs/content/code-review/review-pr.md) and [`.agents/knowledge/index.md`](../../../knowledge/index.md) before judging any change, opening every knowledge entry that matches the touched surface (plus [`schema-review`](../schema-review/SKILL.md) when schemas change). Those documents hold the authoritative review criteria; this skill defines how to gather evidence and separate merge blockers from non-blocking notes without duplicating those rules.

## Prerequisites & Workspace Hygiene

- Follow [`AGENTS.md`](../../../../AGENTS.md) parallel-work rules: keep the repository clean and read-only, write any temporary or parsed-log files outside the repository working tree, and do not post external GitHub review comments unless explicitly asked.
- If inspecting an older PR branch in a temporary worktree, read `.agents/` skills and knowledge from your primary `magic-modules` workspace so guidance stays current.

## Execution Steps

### 1. Gather Full Context & Run Applicable Verification Skills
- Inspect the diff alongside **full surrounding files**—never judge from diff hunks alone. Trace changed fields end-to-end across `product.yaml`, the resource YAML, paired `*Version.yaml` schemas, custom `.go.tmpl` / Go hooks, `*_test.go` / `*.tf.tmpl` test steps, linked issues, and available CI/VCR or debug logs.
- Optional utility skills (invoke selectively when needed and not already covered by CI; never run as a mandatory checklist):
  - [`run-pre-gen-checks`](../run-pre-gen-checks/SKILL.md) for formatting, YAML/template guards, and `mmv1`/tool unit tests.
  - [`validate-provider-changes`](../validate-provider-changes/SKILL.md) for breaking schema changes, missing acceptance tests, and missing docs across GA and Beta.
  - [`parse-debug-logs`](../parse-debug-logs/SKILL.md) (outputting to a temporary directory outside the repo) whenever `TF_LOG=DEBUG` logs are available to inspect actual `POST`/`PATCH`/`GET`/`DELETE` payloads and `updateMask` parameters.
  - [`verify-api-schema`](../verify-api-schema/SKILL.md) when REST Discovery field names, types, or endpoint availability need confirmation.

### 2. Audit Exhaustively Against `review-pr.md` & `.agents/knowledge/`
- Open all matching topic guides from [`.agents/knowledge/index.md`](../../../knowledge/index.md) (field modeling, enums, zero-value clearing, permadiffs, import-safe flatteners, sibling schema sync, templates, test adequacy, and unit-test scope).
- Do not stop after finding the first bug: audit every touched resource, property, custom Go hook, and test step, reporting each distinct root cause separately.
- In addition to compiler and knowledge-base checks, verify that `type: Enum` field descriptions do **not** manually include `Possible values:` lists (MMv1 appends `Possible values: [...]` automatically).

### 3. Apply the Evidence Gate & Report
- **Blocking Findings:** Cite exact file/line locations and concrete code, schema, API, or log evidence. Never raise speculative *"verify whether..."* questions or valid repository conventions documented in `review-pr.md` / `.agents/knowledge/` as blockers.
- **Non-Blocking Observations:** Keep style/readability nits and missing manually pasted GA test output in a GitHub PR description strictly non-blocking when acceptance test configurations exist and checks pass. **Never** demote failing or skipped tests, missing test configurations for added configurable fields, or missing 2-step update coverage for mutable fields to non-blocking.
- Return a concise Markdown report with **Review Summary**, **Blocking Findings** (omit if none), **Non-Blocking Observations**, and **Verdict** (`APPROVE` if zero blocking findings, otherwise `REJECT`).
