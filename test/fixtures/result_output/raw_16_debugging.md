---
provider: github
mode: issues
mode_adjusted: false
truncated: false
sources_count: 10
cached: false
---

**tenstorrent/tt-inference-server#1671** CI: PR comments fail for PRs from forked repositories (open)
  Updated: 2026-01-08
  When a pull request is opened from a forked repository, the "Comment on PR" steps in the test-gate.yml workflow fail with the following error:
  
  ```
    RequestError [HttpError]: Resource not accessible by integration
    status: 403
    'x-accepted-github-permissions': 'issues=write; pull_requests=write'
  ```
  
  This error was at https://github.com/tenstorrent/tt-inference-server/actions/runs/20806082046
  
  For security reasons, GitHub restricts GITHUB_TOKEN permissions for workflows triggered by pull_requ...

**MishaKav/pytest-coverage-comment#243** [Bug]: `Permission denied when trying to create/update comment` from forks (open)
  Labels: bug | Updated: 2026-02-19
  ### What's the bug?
  
  Creating the pytest coverage comment works for branches. Now a PR was created from a fork and the workflow failed with `Error: Permission denied when trying to create/update comment.`
  
  Is there something I am supposed to change?
  
  The workflow output can be seen here: <https://github.com/ctu-geoforall-lab/cnn-lib/actions/runs/22115713625/job/63962336703?pr=36>
  
  A previous run that worked can be seen here: <https://github.com/ctu-geoforall-lab/cnn-lib/actions/runs/21818085850/...

**commit-check/commit-check-action#143** `Error posting PR comment: Resource not accessible by integration: 403` (open)
  Labels: bug | Comments: 5 | Updated: 2025-11-17
  ### describe your issue
  
  While trying the commit-check action with the experimental `pr-comments` option I ran into an issue. 
  
  When using more or less the config example from the [marketplace](https://github.com/marketplace/actions/commit-check-action#usage) the workflow aborted. These are the lines from the log:
  
  ```
  2025-09-11T13:53:02.4577894Z Request GET /repos/csidirop/actiontest/issues/1 failed with 403: Forbidden
  2025-09-11T13:53:02.4580746Z Error posting PR comment: Resource not accessi...

**amannn/action-semantic-pull-request#249** Error: Resource not accessible by integration (open)
  Comments: 3 | Updated: 2024-03-26
  **Describe the bug**
  
  When the action runs, I get the following error message:
  
  >Error: Resource not accessible by integration
  
  I use the following job declaration, essentially comming from the docs:
  
  <details><summary>Display job</summary>
  
  ```
  name: Check PR title
  
  permissions:
    pull-requests: write
  
  on:
    pull_request:
      types:
        - opened
        - edited
        - synchronize
  
  jobs:
    main:
      name: Validate PR title
      runs-on: ubuntu-latest
      steps:
       ...

**marocchino/sticky-pull-request-comment#930** - Resource not accessible by integration (open)
  Comments: 11 | Updated: 2023-08-15
  I've tried omitting the GITHUB_TOKEN, I've tried creating a custom personal access token with every possible permission yet I'm still getting this error: - Resource not accessible by integration
  
  What could possibly be the issue?
  
  ```
  name: Write deploy comment
  on:
    pull_request_target:
      types: [ opened, reopened ]
      branches-ignore:
          - 'production'
  
  jobs:
    trigger:
      name: Write comment with url
      runs-on: ubuntu-latest
      steps:
        - name: Set dev folder...

**ScaCap/action-surefire-report#31** Resource not accessible by integration when using from fork PRs (open)
  Comments: 10 | Updated: 2023-11-06
  Getting this error here when this action runs from a PR from a **fork**:
  ```
  Posting status 'completed' with conclusion 'failure' to https://github.com/microsoft/vscode-python/pull/14326 (sha: f4e60b0f743a056b5bfdfe4c85388eeff145b22e)
  Error: Resource not accessible by integration
  ```
  
  I believe that's because of this:
  https://docs.github.com/en/free-pro-team@latest/actions/reference/authentication-in-a-workflow#permissions-for-the-github_token
  
  Fork PRs don't get write access.
  
  Is the...

**pascalgn/automerge-action#171** Failed to merge PR: Resource not accessible by integration (open)
  Labels: bug | Comments: 7 | Updated: 2024-03-21
  ## Description
  Running into an issue where the action intermittently is unable to merge the pull request. It complains about the resource not being accessible from the integration (`Failed to merge PR: Resource not accessible by integration`).
  
  ### Setup
   - Pull Requests are coming from forks, but that shouldn't matter as we are using the `pull_request_target` event. 
   - We run an auto approve step and then an auto merge. The auto merge ends up failing all retries and returns the following....

**axel-op/dart-package-analyzer#2** No report on pull requests from forks (open)
  Comments: 7 | Updated: 2021-04-19
  ## Description
  When the action is triggered by the `pull_request` event, if the pull request is from a different repository (a fork), the action fails with the error "Resource not accessible by integration".
  
  EDIT: this behavior has changed. See the comment below.
  
  ## Why is this happening
  This action uses the [`GITHUB_TOKEN`](https://help.github.com/en/actions/automating-your-workflow-with-github-actions/authenticating-with-the-github_token) you provide to call the GitHub API and to post ...

**vm0-ai/vm0#2606** feat(ci): enable full CI for external PRs with maintainer approval (open)
  Labels: priority: high | Comments: 1 | Updated: 2026-02-08
  ## Problem
  
  External PRs (from forks) cannot run the full CI pipeline because:
  
  1. `GITHUB_TOKEN` is read-only for fork PRs (GitHub security policy)
  2. Secrets (`NEON_API_KEY`, `VERCEL_TOKEN`, etc.) are not exposed
  3. Cannot create GitHub Deployments
  
  This means external contributors only see lint/test results, but deploy and e2e tests are skipped until merge queue.
  
  **Current behavior:**
  ```
  External PR submitted
      ↓
  ✅ lint, type-check, format, knip
  ✅ test-web, test-cli, test-platform
  �...

**devmasx/merge-branch#19** Does this work with PRs from forks?  (open)
  Comments: 3 | Updated: 2023-04-30
  I'd like to implement this in our "read the docs" repository where contributors contribute PRs from their personal (public) forks. My goal is to auto merge labeled PRs into a development branch so that Read the Docs will auto-build the latest developer docs to make it easy to view content/changes introduced by PRs. 
  
  But I keep getting this error:
  
  ```
  /usr/local/bundle/gems/octokit-4.14.0/lib/octokit/response/raise_error.rb:16:in `on_complete': POST https://api.github.com/repos/mautic/deve...