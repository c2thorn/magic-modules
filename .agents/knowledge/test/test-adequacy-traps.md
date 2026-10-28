---
name: test-adequacy-traps
description: Avoid false-positive acceptance tests from broken CheckDestroy templates, unmutated update steps, or ExpectError-only steps.
topics: [test, custom-code]
task_types: [field-add, new-resource, bug-fix, test-fix]
source: docs/content/test/test.md + mmv1/templates/terraform/samples/base_configs/test_file.go.tmpl
status: draft
last_verified: 2026-10-28
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

## 2. Multi-Step Update Coverage Traps

- **Identical HCL across steps does not test `Update`:** Including a mutable field with the **same** value in Step 1 (`_basic`) and Step 2 (`_update`) results in a no-op plan for that field—neither `d.HasChange` nor `updateMask` is exercised. Every newly added mutable (`!ForceNew`) field must transition between distinct values across two applied `Config` steps (including non-zero -> zero where clearing is supported).
- **Adding vs. modifying vs. clearing:** When possible, transition `unset -> value1 -> value2` or `value1 -> value2 (or zero)` so both `Create`/`Read` and `Update` (`PATCH`) paths run against the live API.

## 3. `ExpectError`-Only Steps Do Not Cover Positive Behavior

- **Rule:** `TestStep` blocks with `ExpectError` (or `PlanOnly: true`) validate negative rejection paths only; they never create, read, update, or import the field in state.
- **Why:** Every configurable field must still appear in a positive applied `Config` step followed by an `ImportStateVerify: true` step.

## Do NOT Use For

- Do **not** write a `custom_check_destroy` template when MMv1's generated `CheckDestroy` already queries the right `self_link` URL—only customize when the API returns `200` with a soft-delete state (for example `state == "DELETED"`), clears a singleton's sub-object, or requires special query parameters on `GET`.
