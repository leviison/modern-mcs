# Branch Protection Guidance

This repository uses a CI workflow at `.github/workflows/ci.yml` with a single job:
- `test-build`

To prevent regressions, configure GitHub branch protection on `main` (or your default branch) with these minimum settings:

1. Require a pull request before merging.
2. Require status checks to pass before merging.
3. Require the status check:
   - `test-build`
4. Require branches to be up to date before merging.
5. Disallow force pushes to protected branches.
6. Restrict who can bypass branch protections (admins optional per policy).

Optional hardening:
- Require at least 1 approving review.
- Dismiss stale reviews when new commits are pushed.
- Require signed commits.
- Require conversation resolution before merging.
