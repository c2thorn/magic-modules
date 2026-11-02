---
name: test-adequacy-traps
description: Avoid false-positive acceptance tests from broken CheckDestroy templates, unmutated update steps, or ExpectError-only steps.
topics: [test, custom-code]
task_types: [field-add, new-resource, bug-fix, test-fix]
source: docs/content/test/test.md + mmv1/templates/terraform/samples/base_configs/test_file.go.tmpl
status: draft
last_verified: 2026-11-02
---

# Acceptance Test Adequacy Traps

## 1. Custom `test_check_destroy` (`custom_check_destroy`) Invariants

`custom_code.test_check_destroy` injects Go code inside the `for name, rs := range s.RootModule().Resources` loop of `testAccCheck<Resource>DestroyProducer`.

- **Rule:** A custom destroy check **must** confirm deletion via HTTP `404` (`transport_tpg.IsGoogleApiErrorWithCode(err, 404)`) or an explicit deleted terminal state on a `200` response, use `continue` when confirmed deleted so remaining resources in state are checked, and `return err` on any unexpected HTTP error (`400`, `403`, `500`).
- **Why:** Checking only `if err == nil { return fmt.Errorf("still exists") }` allows a `400 Bad Request` (such as a malformed URL, unexpanded `{{var}}`, or missing query parameter) to return `err != nil`, silently fall through, and pass `CheckDestroy` while the live cloud resource is leaked. Using `return nil` instead of `continue` skips checking any subsequent resources in `s.RootModule().Resources`.
- **Never** remove eventual-consistency retry loops (`transport_tpg.Retry`) from a destroy check just to make a slow teardown pass faster.

```go
res, err := transport_tpg.SendRequest(transport_tpg.SendRequestOptions{
	Config:    config,
	Method:    "GET",
	Project:   billingProject,
	RawURL:    url,
	UserAgent: config.UserAgent,
})
if err != nil {
	if transport_tpg.IsGoogleApiErrorWithCode(err, 404) {
		continue
	}
	return err
}
return fmt.Errorf("ExampleResource still exists at %s: %v", url, res)
```

## 2. Multi-Step Update Coverage & Scope

- **2-Step Update Rule:** Every newly added mutable (`!ForceNew`) field must be exercised in an update step with `ImportStateVerify: true`—either by transitioning between two distinct values (`valA -> valB`) or by introducing the field in the update step (`unset` in Step 1 `-> valA` in Step 2) so Step 2 sends a `PATCH` with the field in `updateMask`. Once an update step exercises `d.HasChange` and `updateMask`, additional combinatorial sub-transition steps (such as `true -> false` toggles, omitting every optional field in a separate step, or negative `ExpectError` steps) are not required.
- **Identical HCL across steps does not test `Update`:** Including a mutable field with the **same** value in Step 1 (`_basic`) and Step 2 (`_update`) results in a no-op plan for that field—neither `d.HasChange` nor `updateMask` is exercised.
- **Extending Existing `TestAcc` Functions:** Extending an existing `TestAcc<Resource>_basic` / `_update` pair (or existing sample steps) to cover a new field is standard and preferred over creating a separate standalone `TestAcc` function, unless the field requires an incompatible resource mode, `ConflictsWith` branch, or distinct prerequisite topology.
- **`ImportStateVerify: true` Covers Attribute Readback:** When a config step is followed by `ImportState: true` and `ImportStateVerify: true` (and the field is not listed in `ImportStateVerifyIgnore`), Terraform automatically re-reads the resource from the API and verifies state equality for every configured attribute. Explicit `resource.TestCheckResourceAttr` assertions are redundant and unnecessary.
- **Read-Only (`output: true`) Fields:** Attributes marked `output: true` (`Computed`-only) cannot be configured in HCL and do not require HCL config blocks or update mutation steps.
- **Maintainer Approval for Skips:** Any skipped or no-op acceptance test step for modified behavior requires explicit maintainer sign-off on the PR rather than unilateral self-exemption.

## 3. `ExpectError`-Only Steps Do Not Cover Positive Behavior

- **Rule:** `TestStep` blocks with `ExpectError` (or `PlanOnly: true`) validate negative rejection paths only; they never create, read, update, or import the field in state.
- **Why:** Every configurable field must still appear in a positive applied `Config` step followed by an `ImportStateVerify: true` step.

## 4. Sweeper Loop Error Continuation

In multi-item test sweepers (`services/<product>/<resource>_sweeper.go` and `mmv1/templates/terraform/sweeper_file.go.tmpl`), sweeper loops iterate over discovered resources to clean them up:

- **Rule:** Per-item deletion errors inside the resource iteration loop must log the failure and `continue` (never `return nil` or abort the function).
- **Why:** Aborting or returning `nil` on the first item error halts cleanup and leaks all subsequent resources in the test project.

## 5. `FirstTestConfig()` `samples[0]` Reuse & HCL Sample Invariants

- **`FirstTestConfig()` Base Fixture:** MMv1 uses `FirstTestConfig()` (`mmv1/api/resource.go`) to select `samples[0]` as the base test configuration for generated IAM tests (`*_iam_test.go`) and datasource tests. Placing a GA-compatible, minimal basic sample at `samples[0]` ensures generated IAM and datasource tests compile and pass in both GA and Beta providers.
- **Resource ID Prefixing:** `resource_id_vars` automatically receives a `tf-test-` (or `tf_test_`) prefix in acceptance tests; verify that generated resource IDs remain within API length limits.
- **Literal Regions in Samples:** HCL sample templates (`mmv1/templates/terraform/examples/*.tf.tmpl`) routinely specify literal supported GCP regions/zones (such as `"us-central1"`) when required by the API. Only declare template variables in `test_env_vars` or `resource_id_vars` when dynamic test environment substitution is required.
- **Resource-Level `launch_stage` Attributes:** Several GCP APIs expose a configurable resource attribute named `launch_stage` (accepting values such as `"BETA"`, `"ALPHA"`, `"GA"`, or `"PRELAUNCH"`). Setting `launch_stage = "BETA"` inside an HCL resource block configures the cloud API payload property—it is not a provider version-gating leak (`google` vs `google-beta`) and is valid in GA tests whenever the `launch_stage` schema field itself is GA.

## 6. Review Scope & Test Execution Invariants

- **Scoped Test Verification:** Verify that all acceptance and unit tests covering modified resources, fields, and shared helpers pass; pre-existing test failures on unrelated resources in package-wide test logs do not block unrelated changes.
- **Offline Schema/Unit Tests:** Unit and schema validation tests (`func Test...` without `TestAcc`) run offline via `go test` without `TF_ACC=1` and do not emit HTTP wire requests in acceptance test logs.
- **Widening Numeric / `max_size` Bounds:** Increasing an existing `max_size` or numeric limit to match upstream API limits is verified by schema definition and standard acceptance tests without requiring quota-exhausting max-scale live resource counts.
- **Prior Review Round Follow-Up:** When reviewing multi-round PRs or patch revisions, verify that earlier reviewer requests were either implemented in subsequent changes or explicitly answered with technical justification.
- **Missing Author Test Output is Non-Blocking:** When a change affects GA or adds acceptance tests without manual test output pasted in the PR description, note the missing test confirmation as a non-blocking comment for the author or maintainer rather than treating the absence of pasted output as a code defect.
- **PR Descriptions & Release Notes:** When reviewing a published GitHub PR description, verify `release-note` entries accurately describe the user-visible behavior change; when reviewing pre-PR patches or diffs before a PR description exists, focus strictly on code, schema, docs, and test correctness.

## Do NOT Use For

- Do **not** write a `custom_check_destroy` template when MMv1's generated `CheckDestroy` already queries the right `self_link` URL—only customize when the API returns `200` with a soft-delete state (for example `state == "DELETED"`), clears a singleton's sub-object, or requires special query parameters on `GET`.
