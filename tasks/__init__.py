from invoke.collection import Collection

from tasks.docker import create_docker, destroy_docker
from tasks.ecs import create_ecs, destroy_ecs
from tasks.eks import create_eks, destroy_eks
from .vm import create_vm, destroy_vm
from .setup import setup

ns = Collection()
ns.add_task(create_vm)      # pyright: ignore [reportGeneralTypeIssues]
ns.add_task(destroy_vm)     # pyright: ignore [reportGeneralTypeIssues]
ns.add_task(create_docker)  # pyright: ignore [reportGeneralTypeIssues]
ns.add_task(destroy_docker) # pyright: ignore [reportGeneralTypeIssues]
ns.add_task(create_eks)     # pyright: ignore [reportGeneralTypeIssues]
ns.add_task(destroy_eks)    # pyright: ignore [reportGeneralTypeIssues]
ns.add_task(create_ecs)     # pyright: ignore [reportGeneralTypeIssues]
ns.add_task(destroy_ecs)    # pyright: ignore [reportGeneralTypeIssues]
ns.add_task(setup)          # pyright: ignore [reportGeneralTypeIssues]