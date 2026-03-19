# Rebase and Conflict Resolution Task

Your task is to rebase the current branch onto main (or origin/main) and resolve any conflicts that arise.

## Steps

1. **Fetch and Rebase**
   ```bash
   git fetch origin
   git rebase origin/main
   ```

2. **Handle Conflicts (if any)**
   - If rebase has conflicts, you will see conflict markers in files
   - Analyze each conflict carefully
   - Resolve conflicts by keeping the correct changes
   - After resolving all conflicts in a file: `git add <file>`
   - Continue rebase: `git rebase --continue`

3. **Test-Driven Conflict Resolution**
   - After resolving conflicts, run the test suite (e.g., `make test`, `go test ./...`, `npm test`)
   - If tests fail:
     a. Analyze the test failures
     b. Fix the code to make tests pass
     c. Re-run tests
     d. Repeat until all tests pass
   - If new conflicts appear during rebase --continue, repeat the resolution process

4. **Completion Criteria**
   - Rebase completes successfully (no more conflicts)
   - All tests pass
   - Code is in a working state

## Important Notes

- Do NOT commit manually - the system will auto-commit your changes
- Focus on preserving the intent of both branches when resolving conflicts
- When in doubt, prefer the changes from the current feature branch over main
- Write a summary of what conflicts were resolved and how to {{.ArtifactDir}}/rebase_summary.md
