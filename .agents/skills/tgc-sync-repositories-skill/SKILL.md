---
name: tgc-sync-repositories-skill
description: Synchronize Magic Modules and TGC repositories to prevent unintended downstream diffs during generation. Use when testing code generation or comparing downstream diffs.
---

# tgc-sync-repositories-skill

When you need to synchronize the Terraform Google Conversion (TGC) repository with a specific branch in Magic Modules (MM), use this skill.

## When to Use This Skill

- Use this when testing code generation (`make tgc`) and comparing diffs in the downstream TGC repo.
- This is helpful to prevent "drift diffs" / "unexpected diffs" caused by MM and TGC being checked out at different points in their respective `main` branch histories.

---

## How to Use It

If the repositories are out of sync, running `make tgc` will generate hundreds of unrelated changes. Follow these steps to align the downstream repository precisely to the point where the current MM branch diverged from upstream.

### 1. Find the True Upstream Base in Magic Modules

Sometimes the local `main` branch will be fine, but often it can be stale depending on the developer's workflow. Always check and determine the true upstream remote. If the upstream remote configuration is ambiguous (e.g., complex fork naming schemes), **ask the user** for clarification before proceeding to ensure accuracy.

1. Ensure you are in the Magic Modules directory.
2. Identify the core upstream remote (e.g., `upstream` if `origin` is a personal fork). You can check via `git remote -v`. Ask the user if you are unsure which remote represents the source of truth.
3. Fetch the upstream remote: `git fetch <upstream-remote> main`
4. Find the true divergence base commit of the current branch against that upstream:
   ```bash
   git merge-base <upstream-remote>/main HEAD
   ```
5. Extract the timestamp, pull request number (e.g., `(#12345)`), and commit message of that base commit by inspecting it (`git log -1 <base-sha>`).

### 2. Find the Corresponding Commit in TGC

1. Navigate to the downstream TGC repository (`terraform-google-conversion`).
2. Identify and fetch its primary upstream remote as you did in MM.
3. Search the upstream tracking branch (`<upstream-remote>/main`) around the same date or using the exact PR number from Phase 1. 
   - *Note*: Generated TGC commits closely follow their origin MM commits, often mirroring the exact same PR number `#...`.
   ```bash
   git log <upstream-remote>/main --grep="<PR_NUMBER>"
   ```
4. Check out that precisely matched downstream commit to align the repositories:
   ```bash
   git reset --hard HEAD && git clean -fd
   git checkout <matched-sha>
   ```
   *(This places TGC in a detached HEAD state at the exact historical point of divergence).*

### 3. Verify Alignment

1. Return to the Magic Modules repository.
2. Run the code generation build process (e.g., using `tgc-build-skill`). 
3. Check `git status` in the TGC repository. The generated diffs should now map **exclusively** to the unique commits added to the MM branch after the base commit. If the branch has no generation-impacting changes, `git status` will be perfectly clean (0 diffs).
