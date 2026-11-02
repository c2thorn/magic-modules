---
title: "Review a pull request"
weight: 20
aliases:
  - /contribute/review-pr
---

# Review a pull request

This page provides guidelines for reviewing a Magic Modules pull request (PR).

Google-driven changes must have a service team engineer review changes prior
to review by repository maintainers. For community members, this is not required.

## Review

1. Read the PR description to understand the context and ensure the PR either
   * is linked to a GitHub issue or an internal bug
      * if not, check the [issue tracker](https://github.com/hashicorp/terraform-provider-google/issues) to see whether the feature has already been requested and add the issues in the description, if any.
      * "Fixes {github_issue_link}" is preferred if an external issue is available because it will [auto-close the issue](https://docs.github.com/en/issues/tracking-your-work-with-issues/using-issues/linking-a-pull-request-to-an-issue) when the PR is merged. However, there's no need to create an external issue solely for this purpose.
   * establishes clear context itself via title or description.
2. If the PR adds any new resource, ensure that the resource does not already exist in the [GA provider](https://github.com/hashicorp/terraform-provider-google) or [beta provider](https://github.com/hashicorp/terraform-provider-google-beta)
1. Read through all the changes in the PR, generated code in the downstreams and the API documentation to ensure that:
   1. the resource schema added in the PR matches the API structure.
   1. the features are added in the correct version
      * features only available in beta are not included in the GA google provider.
      * features added to the GA provider are also included in the beta provider -- beta should be a strict superset of GA.
      * distinguish provider version gating (`min_version: beta` / `provider = google-beta`) from resource-level API attributes named `launch_stage` (such as `launch_stage = "BETA"` in HCL, which configures the cloud resource's own stage property in the API payload and is valid in GA tests whenever the field itself is GA).
   1. no [breaking changes]({{< ref "/breaking-changes/make-a-breaking-change" >}}) are introduced without a valid justification. Add the `override-breaking-change` label if there is a valid justification.
      * remember to check for changes in default behaviour like changing the flags on delete! 
   1. verify the change **fully** resolves the linked issues, if any. If it does not, change the "Fixes" message to "Part of".
   1. on follow-up review rounds, verify that prior reviewer comments or requested changes were either implemented in subsequent commits or explicitly answered with technical justification.
   1. inline dependency bumps in `go.mod`/`go.sum` (across provider, `mmv1/go.mod`, and `tools/**/go.mod`) as well as repository metadata updates in `tools/issue-labeler/labeler/enrolled_teams.yml` and `autogen_status` headers are valid in feature and bugfix PRs whenever compilation and checks pass.
1. Check the tests added/modified to ensure that:
   1. all fields added/updated in the PR appear in at least one test. Extending an existing `TestAcc<Resource>_basic` / `_update` pair (or existing sample steps) to include new fields is preferred over creating a redundant standalone test function unless the field requires a conflicting resource mode or distinct topology.
      * It is advisable to test updating from a non-zero value to a zero value if feasible.
   1. all mutable fields are tested in at least one 2-step update test with `ImportStateVerify: true`—either transitioning between two distinct values (`valA -> valB`) or introducing the field in the update step (`unset` in step 1 `-> valA` in step 2) so the update step exercises `PATCH` and `updateMask`. Once an update step passes cleanly, additional combinatorial sub-transition steps (such as `true -> false` toggles or negative `ExpectError` steps) are not required.
   1. fields marked `output: true` (`Computed: true` with `Optional: false, Required: false`) cannot be configured or mutated in HCL and do not require HCL configuration blocks or update mutation steps.
   1. all resources in the acceptance tests have a `tf-test` or `tf_test` prefix in their primary id field.
   1. all handwritten test Config steps include import steps (`ImportState: true` and `ImportStateVerify: true`) following them. Because `ImportStateVerify: true` automatically re-reads the resource from the API and verifies state equality for all configured attributes, explicit `resource.TestCheckResourceAttr` assertions are unnecessary unless a field is excluded from import verification.
   1. all related tests pass in GA for features promoted from beta to GA.
      > [!NOTE]
      > Note:
      > Presubmit VCR tests do not run in GA. Manual testing is required for promoted GA features. When manual GA test output is not pasted in the PR description, ask the author or maintainer to confirm or attach test results in a non-blocking comment rather than treating the missing comment as a code defect.
   1. custom helper functions (`DiffSuppressFunc`, parsers, expanders/flatteners) containing multi-branch logic, loops, string/regex parsing, or algorithmic transformations are tested in at least one unit test. Attaching existing shared helpers (`tpgresource.*`, `customdiff.*`), writing trivial (`<= 5` lines, single-condition) custom `DiffSuppressFunc`s exercised by acceptance tests, or writing `CustomizeDiff` functions that rely on unmockable `schema.ResourceDiff` methods (`GetRawConfig`) do not require standalone unit tests and are verified through acceptance tests.
   1. unit and schema validation tests (`func Test...` without the `TestAcc` prefix) execute offline via `go test` without `TF_ACC=1` and do not emit HTTP requests in acceptance wire logs.
   1. when increasing an existing `max_size` or numeric validation limit to match upstream API limits, verify the schema definition and standard acceptance test without requiring quota-exhausting max-scale live resource counts.
   1. when deprecating a field in a minor release (without removing it yet), existing acceptance tests that exercise the deprecated field remain intact until major-release removal, with separate test coverage added for any replacement field.
   1. the linked issue (if any) is covered by at least one test that reproduces the issue
      * for example - a bugfix should test the bug (or explain why it's not feasible to do so in the description, including manual results when possible) and an enhancement should test the new behaviour(s).
   1. all related PR presubmit tests have been completed successfully, including:
      * terraform-provider-breaking-change-test
      * terraform-provider-google-build-and-unit-tests
      * terraform-provider-google-beta-build-and-unit-tests
      * VCR-test
      > [!NOTE]
      > Note:
      > Any skipped or no-op acceptance test step for modified behavior requires explicit maintainer sign-off on the PR rather than unilateral self-exemption.
   1. all acceptance and unit tests covering the resources, fields, and shared helpers modified by the PR pass; pre-existing test failures on unrelated resources in package-wide test logs do not block unrelated changes.
   1. a significant number of preexisting tests have not been modified. Changing old tests often indicates a change is backwards incompatible.
1. Check documentation to ensure
   1. resource-level and field-level documentation are generated correctly for MMv1-based resource
   1. documentation is added manually for handwritten resources.   
1. Check release notes and metadata:
   1. when reviewing a published GitHub PR description, verify that `release-note` entries accurately describe the user-visible behavior change following the guidance in [write release notes]({{< ref "release-notes" >}}). When reviewing pre-PR patches or diffs before a PR description exists, focus strictly on code, schema, docs, and test correctness.

If any of these steps are impossible, clearly explain that in a comment on the PR or in a summary before handing off to a maintainer for final review.

## Special cases

The following types of PRs may require additional scrutiny and/or multiple reviewers.

- DCL to MMv1 migrations
- Adding multi-actor fields (fields whose values can be altered as a side effect of changes made to a different resource)