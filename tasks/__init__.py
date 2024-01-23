from invoke.collection import Collection

from tasks.aks import create_aks, destroy_aks
from tasks.docker import create_docker, destroy_docker
from tasks.ecs import create_ecs, destroy_ecs
from tasks.eks import create_eks, destroy_eks

from .setup import setup
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
ns.add_task(setup)  # pyright: ignore [reportArgumentType]
