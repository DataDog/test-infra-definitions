from invoke import Collection
from .vm import vm
from .destroy import destroy

ns = Collection()

deploy = Collection("create")
deploy.add_task(vm)

ns.add_collection(deploy)
ns.add_task(destroy)