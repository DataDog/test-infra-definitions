import os
import re
from io import StringIO
from invoke.tasks import task

from github import Github, Auth
from invoke.context import Context
from invoke.exceptions import Exit
from termcolor import colored


COMMIT_TITLE_REGEX = re.compile(r"\[test-infra-definitions\]\[automated\] Bump test-infra-definitions to ([a-z0-9]*)")

@task
def create_pr_and_close_stale_ones(ctx, branch: str, new_commit_sha: str, old_commit_sha: str):
    # if os.getenv("CI") != "true":
    #     print("This task should only be run in CI")
    #     return
    # Create a PR
    if os.getenv("GITHUB_TOKEN") is None:
        print("GITHUB_TOKEN is not set")
        return
    
    repo = Github(auth=Auth.Token(os.environ["GITHUB_TOKEN"])).get_repo("DataDog/datadog-agent")
    pr_body = f"""
This PR was automatically created by the test-infra-definitions bump task.

This PR bumps the test-infra-definitions submodule to {new_commit_sha} from {old_commit_sha}.
Here is the full changelog between the two commits: https://github.com/DataDog/test-infra-definitions/compare/{old_commit_sha}..{new_commit_sha}

:warning: This PR is opened with the `qa/no-code-change` and `changelog/no-changelog` labels by default. Please make sure this is appropriate
    """

    new_pr =repo.create_pull(title=f"[test-infra-definitions][automated] Bump test-infra-definitions to {new_commit_sha}", body=pr_body, head=branch, base="main")
    new_pr.add_to_labels("qa/no-code-change", "changelog/no-changelog", "automatic/test-infra-bump")

    print(f"PR created: {new_pr.html_url}")

    print("Closing stale auto bump PRs...")

    issues = repo.get_issues(state="open", labels=["automatic/test-infra-bump"])
    prs = [issue.as_pull_request() for issue in issues if issue.pull_request is not None and issue.number != new_pr.number]
    for pr in prs:
        pr_commit_sha_match = re.search(COMMIT_TITLE_REGEX, pr.title)
        if pr_commit_sha_match is None:
            print(f"No commit sha found in PR title:", pr.html_url)
            continue
        pr_commit_sha = pr_commit_sha_match.group(1)
        res = ctx.run(f'git merge-base --is-ancestor {pr_commit_sha} {new_commit_sha}', warn=True, hide="both")
        if res.exited != 0:
            print (f"Commit {pr_commit_sha} is not considered stale, skipping...")
            continue
        reviews = pr.get_reviews()
        if reviews.totalCount != 0:
            print(f"PR {pr.html_url} has reviews, skipping...")
            continue
        print("Closing PR: ", pr.html_url)
