import os
from datetime import datetime

import gitlab
from dateutil import parser
from invoke.tasks import task

from . import doc


@task(
    help={
        'pipeline-id': doc.pipeline_id,
        'job-name': doc.job_name,
    }
)
def retry_job(_, pipeline_id, job_name):
    """
    Retry gitlab pipeline job
    """
    agent, _, job = _get_job(pipeline_id, job_name)

    print(f'Retrying job {job_name} ({job.id})...')
    new_job = agent.jobs.get(job.id).retry()
    print(
        f'Job {job_name} retried, see status at https://gitlab.ddbuild.io/DataDog/datadog-agent/-/jobs/{new_job["id"]}'
    )


def get_job_time_since_execution(pipeline_id, job_name) -> float:
    """
    Retrieves the job time since execution in hours
    """
    _, _, job = _get_job(pipeline_id, job_name)
    finished_at = parser.isoparse(job.finished_at)

    diff = datetime.now(finished_at.tzinfo) - finished_at
    diff_hours = diff.total_seconds() / (60 * 60)

    return diff_hours


def _get_job(pipeline_id, job_name):
    """
    Get gitlab job of DataDog/datadog-agent from pipeline and name
    Returns (repository, pipeline, job)
    """
    gitlab_token = os.environ['GITLAB_TOKEN']

    gl = gitlab.Gitlab('https://gitlab.ddbuild.io', private_token=gitlab_token)
    agent_repo = gl.projects.get('DataDog/datadog-agent')
    pipeline = agent_repo.pipelines.get(pipeline_id)

    # TODO : Optimize (graphql / filter by stage ?)
    jobs = pipeline.jobs.list(all=True, per_page=100)

    # Latest job first by default
    job = [j for j in jobs if j.name == job_name]
    assert len(job) >= 1, f'Cannot find job {job_name}'
    job = job[0]

    return agent_repo, pipeline, job
