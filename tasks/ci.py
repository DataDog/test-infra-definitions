import os
from io import StringIO
from invoke.tasks import task

from github import Github, Auth
from invoke.context import Context
from invoke.exceptions import Exit
from termcolor import colored


@task
def create_pr(ctx, branch: str, new_commit_sha: str, old_commit_sha: str):
    # if os.getenv("CI") != "true":
    #     print("This task should only be run in CI")
    #     return
    # Create a PR
    if os.getenv("GITHUB_TOKEN") is None:
        print("GITHUB_TOKEN is not set")
        return
    
    repo = Github(auth=Auth.Token(os.environ["FAKE_TOKEN"])).get_repo("DataDog/datadog-agent")
    pr_body = f"""
    This PR was automatically created by the test-infra-definitions bump task.
    
    This PR bumps the test-infra-definitions submodule to {new_commit_sha} from {old_commit_sha}.
    Here is the full changelog between the two commits: https://github.com/DataDog/test-infra-definitions/compare/{old_commit_sha}..{new_commit_sha}

    :warning: This PR is opened with the `qa/no-code-change` and `changelog/no-changelog` labels by default. Please make sure this is appropriate
    """

    new_pr =repo.create_pull(title=f"[test-infra-definitions][automated] Bump test-infra-definitions to {new_commit_sha}", body=pr_body, head=branch, base="main", draft=True)
    new_pr.add_to_labels("qa/no-code-change", "changelog/no-changelog", "automatic/test-infra-bump")
    print("Bumping test-infra on datadog-agent")
