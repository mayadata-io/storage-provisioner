## Running on command line

For debugging, it's possible to run the storage-provisioner on command line:

```sh
# cd to root of this project
make

dao-storprovisioner -kubeconfig ~/.kube/config -v 5
```

## Implementation details

The storage-provisioner follows [controller](https://github.com/kubernetes/community/blob/master/contributors/devel/controllers.md) pattern and uses informers to watch for `Storage` and `PersistentVolumeClaim` create/update/delete events.

## Troubleshooting
