from invoke import Collection

from tasks.docker import create_docker, destroy_docker
from tasks.eks import create_eks, destroy_eks
from .vm import create_vm, destroy_vm

ns = Collection()
ns.add_task(create_vm)
ns.add_task(destroy_vm)
ns.add_task(create_docker)
ns.add_task(destroy_docker)
ns.add_task(create_eks)
ns.add_task(destroy_eks)