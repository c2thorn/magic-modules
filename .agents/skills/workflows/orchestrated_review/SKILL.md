---
name: orchestrated-review-workflow
description: "Opt-in two-wave multi-agent code review workflow (holistic Lead pass + 3 parallel read-only domain specialists). Invoke ONLY when the user explicitly asks for an orchestrated or multi-agent review; use code-review for standard reviews."
---

# `orchestrated-review-workflow`

> **Note to AI Agents:** You MUST read the YAML frontmatter above first. Run this workflow **only** when the user explicitly requests an orchestrated or multi-agent review. For standard PR, branch, or diff reviews—or when using a long-horizon reasoning model where a single holistic pass holds full cross-file context without multi-agent token overhead—use [`.agents/skills/utils/code-review/SKILL.md`](../../utils/code-review/SKILL.md) directly.

This opt-in workflow scales [`.agents/skills/utils/code-review/SKILL.md`](../../utils/code-review/SKILL.md) for large multi-resource changes, Beta-to-GA promotions, or complex custom-Go PRs: the main agent runs shared pre-checks once, performs a holistic end-to-end pass in parallel with three read-only [`code-reviewer`](../../../agents/code-reviewer/agent.json) subagent slices, and verifies every specialist lead directly against repository files before synthesizing one deduplicated report.

## Execution Steps

### 1. Run Shared Context & Pre-Checks Once
Complete Step 1 of [`code-review`](../../utils/code-review/SKILL.md) once in a temporary directory outside the repository (`git diff`, touched file list, CI/issue context, and—when not already covered by passing CI—[`run-pre-gen-checks`](../../utils/run-pre-gen-checks/SKILL.md), [`validate-provider-changes`](../../utils/validate-provider-changes/SKILL.md), [`parse-debug-logs`](../../utils/parse-debug-logs/SKILL.md), or [`verify-api-schema`](../../utils/verify-api-schema/SKILL.md)) so specialist subagents read shared outputs instead of repeating builds or log parsing.

### 2. Wave 1 — Parallel Holistic Pass + 3 Read-Only Domain Slices
Launch three parallel read-only [`code-reviewer`](../../../agents/code-reviewer/agent.json) subagents (via `invoke_subagent` with `"code-reviewer"`, or your environment's equivalent read-only subagent tool) while performing your own holistic end-to-end trace across `product.yaml` / resource YAMLs $\to$ custom `.go.tmpl` / Go hooks $\to$ tests and logs.

Pass each subagent the paths to the Step 1 shared outputs, instruct it to read [`code-review/SKILL.md`](../../utils/code-review/SKILL.md), [`docs/content/code-review/review-pr.md`](../../../../docs/content/code-review/review-pr.md), and matching entries in [`.agents/knowledge/index.md`](../../../knowledge/index.md) from the primary `magic-modules` workspace, and assign one domain slice per subagent:

1. **`schema-contract`:** API field names (`api_name`), types, `required`/`output` flags, and URLs; `min_version` GA/Beta placement and GA-promotion completeness; sibling `*Version.yaml` parity and TGC/CAI flags (`is_missing_in_cai`); and enum conventions (`Enum` vs `String`, `*_UNSPECIFIED` handling, non-empty `enum_values`, and no duplicate manual `Possible values:` in `type: Enum` descriptions).
2. **`lifecycle-custom-code`:** Nested `immutable: true` propagation on updatable MMv1 resources and `url_param_only` identity fields; dead `timeouts.update_minutes` on immutable resources; permadiff handling (`default_from_api`, `diff_suppress_func`, `ignore_read` vs `write_only`) and zero-value clearing (`send_empty_value`, `allow_empty_object`); import-safe flatteners; custom Go hooks/encoders/decoders; and backward compatibility.
3. **`test-adequacy`:** Acceptance test and wire-payload coverage for every added configurable field; 2-step update (`PATCH`/`PUT`) coverage for mutable fields; skipped or deleted test coverage and `ImportStateVerifyIgnore` additions; `CheckDestroy` and sweeper error continuation; and unit-test scope for multi-branch custom helpers.

Require each specialist to audit every item in its slice without stopping early and return:
- **Scope & Docs Read**
- **Audit Ledger:** per-resource/field/file entries (`target`, `check`, `PASS` | `DEFECT` | `LEAD`)
- **Verified Blockers:** exact `file:line`, concrete evidence, and impact
- **Leads Needing Cross-File Check:** cross-file hypotheses listing exact files to inspect
- **Non-Blocking Observations**

### 3. Wave 2 — Direct Verification, Deduplication & Synthesis
1. Read every repository file cited in **Leads Needing Cross-File Check** directly before accepting or dismissing any lead.
2. Retain 100% of distinct verified blocking defects; merge duplicate reports that share the same underlying root cause into a single finding with combined field lists and strongest `file:line` evidence.
3. Apply Step 3 of [`code-review`](../../utils/code-review/SKILL.md) so speculative claims stay out of blockers and non-blocking notes never mask failing/skipped tests or missing required test coverage.
4. Confirm `git status` is clean in `magic-modules` and present the synthesized report (**Review Summary**, **Blocking Findings**, **Non-Blocking Observations**, and **Verdict**).
