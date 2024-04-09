from invoke.collection import Collection

import tasks.ci as ci
import tasks.setup as setup
import tasks.test as test
from tasks.aks import create_aks, destroy_aks
from tasks.deploy import check_s3_image_exists
from tasks.docker import create_docker, destroy_docker
from tasks.ecs import create_ecs, destroy_ecs
from tasks.eks import create_eks, destroy_eks
from tasks.installer import create_installer_lab, destroy_installer_lab
from tasks.kind import create_kind, destroy_kind
from tasks.pipeline import retry_job

from .vm import create_vm, destroy_vm

ns = Collection()
ns.add_task(create_vm)  # pyright: ignore [reportArgumentType]
ns.add_task(destroy_vm)  # pyright: ignore [reportArgumentType]
ns.add_task(create_docker)  # pyright: ignore [reportArgumentType]
ns.add_task(destroy_docker)  # pyright: ignore [reportArgumentType]
ns.add_task(create_eks)  # pyright: ignore [reportArgumentType]
ns.add_task(destroy_eks)  # pyright: ignore [reportArgumentType]
ns.add_task(create_aks)  # pyright: ignore [reportArgumentType]
ns.add_task(destroy_aks)  # pyright: ignore [reportArgumentType]
ns.add_task(create_ecs)  # pyright: ignore [reportArgumentType]
ns.add_task(destroy_ecs)  # pyright: ignore [reportArgumentType]
ns.add_task(create_kind)  # pyright: ignore [reportArgumentType]
ns.add_task(destroy_kind)  # pyright: ignore [reportArgumentType]
ns.add_task(retry_job)  # pyright: ignore [reportArgumentType]
ns.add_task(check_s3_image_exists)  # pyright: ignore [reportArgumentType]
ns.add_collection(setup)  # pyright: ignore [reportArgumentType]
ns.add_collection(test)  # pyright: ignore [reportArgumentType]
ns.add_collection(ci)  # pyright: ignore [reportArgumentType]
ns.add_task(create_installer_lab) # pyright: ignore [reportArgumentType]
ns.add_task(destroy_installer_lab) # pyright: ignore [reportArgumentType]
