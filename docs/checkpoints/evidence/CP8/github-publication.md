# GitHub publication verification

28 October 2026. User explicitly selected personal GitHub account `git-patra`, completed login, and requested a public repository and push to `main`.

- Repository: https://github.com/git-patra/patra-world-social
- API-verified owner: `git-patra`; visibility: `PUBLIC`; default branch: `main`.
- Initial public root commit: `0adf204d59ddfd47eeaaa3ca70d8d1ffc0fbf6f9`, verified against the remote `refs/heads/main` after push.
- That root has no parents and exactly matches the tree at local publication-preparation commit `9d0dd25`. Application, contracts, scripts and configuration match verified implementation `58d35d0`; only publication documentation changed afterward.
- Approved design, implementation plan, regression tests, backend verification, fresh review, and browser screenshots are tracked in the public snapshot. Private environment/database/cache/build files and the inherited CV/PDF history are excluded.
- This documentation follow-up records publication completion; it makes no application changes and does not claim a new test run. Existing test evidence and acceptance limits remain unchanged.
- Local active branch is `main`, tracking `origin/main`. The inherited `feat/social-world` branch is preserved locally and was not pushed. No force push or history rewrite was performed on the remote.

## Authentication and transport

CLI login was verified as `git-patra`. The machine's global HTTPS rewrite initially selected the work SSH identity and was rejected. An HTTPS retry authenticated correctly but OAuth lacked permission to upload the existing CI workflow. The already-configured personal SSH identity was then independently verified as `git-patra`; push succeeded through that identity, including `.github/workflows/ci.yml`. No OAuth scope expansion was needed. Temporary repository-local HTTPS/helper overrides were removed, and global Git/SSH configuration was left unchanged.

## Limits

Repository publication is complete; public application deployment was not performed. Account/multi-browser acceptance, LiveKit staging, physical-device acceptance and the remaining T14/T15/T16 gates retain their existing status. CP1 is still the only official PASS.
