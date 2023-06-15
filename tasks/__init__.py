from invoke.collection import Collection
from invoke.tasks import Task

from tasks.docker import create_docker, destroy_docker
from tasks.ecs import create_ecs, destroy_ecs
from tasks.eks import create_eks, destroy_eks
from .vm import create_vm, destroy_vm
from .setup import setup

ns = Collection()
ns.add_task(Task(create_vm))
ns.add_task(Task(destroy_vm))
ns.add_task(Task(create_docker))
ns.add_task(Task(destroy_docker))
ns.add_task(Task(create_eks))
ns.add_task(Task(destroy_eks))
ns.add_task(Task(create_ecs))
ns.add_task(Task(destroy_ecs))
ns.add_task(Task(setup))