from invoke import Collection
from .vm import create_vm, destroy_vm

ns = Collection()
ns.add_task(create_vm)
ns.add_task(destroy_vm)