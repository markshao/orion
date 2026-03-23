---
name: push-remote
description: Commit and push git changes with an integration-first workflow. Use when the user invokes `/push_remote`, explicitly asks to commit and push code, or wants Codex to format the current branch's code, create a concise commit, rebase onto `main`, resolve conflicts with preference for the current feature branch behavior, run the existing test suite without modifying tests, and push the branch safely.
---

# Push Remote

## Overview

Review the current git state, normalize the branch with the repository's formatter, produce a defensible commit, rebase onto `main`, validate by running the existing tests, and push without disturbing unrelated work. Prefer a conservative workflow: understand what changed first, commit only the intended files, preserve feature-branch intent during conflict resolution, and stop when the repository state is ambiguous or risky.

## Workflow

1. Confirm the repository state.
Run `git status --short --branch` first. Determine the current branch, whether an upstream is configured, and whether there are staged, unstaged, or untracked files.

2. Exit early only when the branch is truly a no-op.
Do not treat an empty staging area as sufficient to stop. If the working tree is clean and there are no new local file changes, still fetch and compare the current branch against both its upstream and `origin/main`. Only stop immediately when all of the following are true:
- the working tree is clean,
- there is nothing staged,
- the branch does not need a new commit,
- the branch is not ahead of its upstream, and
- rebasing onto `origin/main` would be a no-op because the branch is already based on the latest `main`.

If the branch is clean but may still require integration work, continue with fetch and rebase. This is important because rebasing onto the latest `origin/main` may surface conflicts that should be resolved before opening or updating an MR.

3. Inspect the actual changes before committing.
Read the staged diff with `git diff --cached`. If there are no staged changes, read `git diff` and decide whether the modified files are obviously part of one coherent change. If the scope is unclear, ask the user before staging anything.

4. Format the branch using the repository's normal formatter.
Infer the language and project tooling from the repository and run the standard formatter before committing. Prefer existing project commands and configuration, such as package scripts, `make fmt`, `cargo fmt`, `gofmt`, `go fmt`, `black`, `ruff format`, `prettier`, or other repo-native tooling. Do not invent a formatter setup that the repository does not already use.

5. Stage intentionally.
If files are already staged, preserve that choice and commit only the staged set unless the user asks otherwise. If nothing is staged, stage only the files that clearly belong to the requested change. Do not blindly use `git add .`.

6. Write a concise commit message from the diff.
Use an imperative subject line. Keep it specific to the behavior change, not the tool used. Add a short body only when the diff contains multiple meaningful effects or non-obvious context.

7. Commit and verify.
Create the commit, then run `git status --short --branch` again to ensure the intended changes were recorded and identify any leftover files.

8. Rebase onto `main` before pushing.
Fetch the latest remote state, update local knowledge of `main`, and rebase the current feature branch onto `origin/main`. Prefer a standard rebase flow rather than merge commits. Do this even when there are no newly staged changes if the branch may still need integration with the latest `main`.

9. Resolve conflicts with feature-branch preference.
Attempt to resolve straightforward textual conflicts directly. When conflict resolution involves a logical choice, keep the current feature branch behavior unless doing so clearly breaks the build or tests. Do not modify test files to make the rebase pass.

10. Run the existing test suite.
Detect the repository's standard test command and run it after the rebase is complete. If multiple common test entry points exist, prefer the project's documented or already-used command. Use the results to guide code fixes, but do not alter tests.

11. Push conservatively.
Push the rebased current branch to its configured upstream when one exists. If no upstream is configured, prefer `git push -u <remote> <branch>` only after confirming the correct remote and branch. If multiple plausible remotes exist, ask the user.

## Commit Rules

- Prefer committing only files that were part of the requested task.
- Preserve unrelated user changes; do not stage or revert them.
- Run the repository's formatter before committing when formatter tooling is discoverable.
- Avoid generated files, secrets, credentials, local environment files, and large lockfile churn unless they are clearly intended.
- If the branch is already clean, only stop early after confirming there is nothing new to commit, nothing new to push, and no rebase needed against the latest `origin/main`.
- Run the existing tests after rebasing and before pushing. Do not edit, relax, skip, or rewrite test code to force a pass.
- Mention formatter and test commands in the final response if you ran them; do not claim validation you did not perform.
- If the diff mixes unrelated concerns, split the work or ask the user before committing.
- If rebase conflict resolution requires a product decision, prefer the feature branch's current logic.

## Push Blocking Conditions

Stop and ask the user instead of pushing when any of the following is true:

- The branch or remote cannot be identified confidently.
- There are no new commits, no working tree changes, and no rebase needed against the latest `origin/main`.
- The working tree contains unrelated modifications that make staging ambiguous.
- The push would include files that look sensitive or accidental.
- The repository is in the middle of a merge, rebase, cherry-pick, or has unresolved conflicts that cannot be resolved confidently.
- The rebase onto `main` fails and the remaining conflicts are too ambiguous to resolve safely.
- The post-rebase tests fail and the fix is not local to the feature branch code.
- The remote branch has diverged and the push may require force or another non-routine history rewrite.
- The user appears to expect review, explanation, or a dry run rather than an actual push.

## Response Pattern

When the workflow succeeds, report:

- The branch pushed
- The commit hash and subject
- That the branch was rebased onto `main`
- The formatter command used, if any
- The test command used and whether it passed
- Whether an upstream was configured or changed
- Any remaining local changes that were intentionally left out

When the workflow stops short, report the exact blocker and the smallest next decision needed from the user.
If the stop reason is a true no-op branch, report that there are no new local changes, no rebase work against `origin/main`, and no push is needed.

## Example Requests

- `/push_remote`
- `Use $push-remote to commit the current work and push it.`
- `Review the git changes, write a clean commit message, and push my branch.`
