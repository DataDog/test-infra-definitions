import os
import os.path
from pathlib import Path

from invoke.tasks import task
from invoke.context import Context

from .destroy import destroy
from .tool import info, is_windows, warn

@task
def clean(_: Context) -> None:
    """
    Clean any environment created with invoke tasks or e2e tests
    """
    info("🧹 Clean up lock files")

    if is_windows():
        warn(f"This task is not supported yet on Windows. Want to help supporting it ? Let's go to {__file__} and be the change !")
        return
    
    lock_dir = os.path.join(Path.home(), ".pulumi", "locks")
    for filename in os.listdir(Path(lock_dir)):
        file_path = os.path.join(lock_dir, filename)
        if os.path.isfile(file_path):
            os.remove(file_path)
            print(f"🗑️ Deleted file: {file_path}")

    info("🧹 Clean up stacks")
    destroy("*")

