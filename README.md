# storage-provisioner
Simplified storage operations using Kubernetes & CSI.

## Roadmap
- GKE
  - DONE: Provision PD via its CSI driver
  - DONE: Attach provisioned PD to a given Kubernetes node
- EKS
  - DONE: Provision EBS disk via its CSI driver
  - DONE: Attach provisioned EBS disk to a given Kubernetes node

## Following steps describe storage provisioner working with GKE Persistent Disk(s)
- Have a Kubernetes setup with minimum version of 1.13.7

```bash
  storage-provisioner > kubectl get node
NAME                                       STATUS   ROLES    AGE     VERSION
gke-amitd-ddp-default-pool-d5aa3f95-ht99   Ready    <none>   6d22h   v1.13.7-gke.24
gke-amitd-ddp-default-pool-d5aa3f95-t8p1   Ready    <none>   6d20h   v1.13.7-gke.24
gke-amitd-ddp-default-pool-d5aa3f95-wq0f   Ready    <none>   6d22h   v1.13.7-gke.24
```

- Deploy google PD CSI driver
- Refer: https://github.com/kubernetes-sigs/gcp-compute-persistent-disk-csi-driver
- Check the compatibility matrix and follow the deploy steps as mentioned in the csi driver repo


- Verify running of csi driver controller

```bash
  storage-provisioner > kubectl get sts
NAME                    READY   AGE
csi-gce-pd-controller   1/1     6d22h


  storage-provisioner > kubectl get sts -owide
NAME                    READY   AGE     CONTAINERS                                   IMAGES
csi-gce-pd-controller   1/1     6d22h   csi-provisioner,csi-attacher,gce-pd-driver   gcr.io/gke-release/csi-provisioner:v1.0.1-gke.0,gcr.io/gke-release/csi-attacher:v1.0.1-gke.0,gcr.io/gke-release/gcp-compute-persistent-disk-csi-driver:v0.4.0-gke.0
```

- Verify running of csi node daemon

```bash
  storage-provisioner > kubectl get daemonset
NAME              DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE
csi-gce-pd-node   3         3         3       3            3           <none>          6d22h
  

  storage-provisioner > kubectl get daemonset -owide
NAME              DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR   AGE     CONTAINERS                           IMAGES                                                                                                                             SELECTOR
csi-gce-pd-node   3         3         3       3            3           <none>          6d22h   csi-driver-registrar,gce-pd-driver   gcr.io/gke-release/csi-node-driver-registrar:v1.0.1-gke.0,gcr.io/gke-release/gcp-compute-persistent-disk-csi-driver:v0.4.0-gke.0   app=gcp-compute-persistent-disk-csi-driver
```

- Verify all the pods to be running

```bash
  storage-provisioner > kubectl get po
NAME                      READY   STATUS    RESTARTS   AGE
csi-gce-pd-controller-0   3/3     Running   0          6d22h
csi-gce-pd-node-6f9rt     2/2     Running   0          6d22h
csi-gce-pd-node-cbb5v     2/2     Running   0          6d22h
csi-gce-pd-node-hkwl8     2/2     Running   0          6d20h
```

- Verify csi driver specific StorageClass

```bash
  storage-provisioner > kubectl get sc
NAME                 PROVISIONER             AGE
csi-gce-pd           pd.csi.storage.gke.io   7d23h
standard (default)   kubernetes.io/gce-pd    8d
```
```yaml
# kubectl get sc csi-gce-pd -oyaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"storage.k8s.io/v1beta1","kind":"StorageClass","metadata":{"annotations":{},"name":"csi-gce-pd"},"parameters":{"type":"pd-standard"},"provisioner":"pd.csi.storage.gke.io","volumeBindingMode":"Immediate"}
  creationTimestamp: "2019-09-11T09:04:58Z"
  name: csi-gce-pd
  resourceVersion: "12161"
  selfLink: /apis/storage.k8s.io/v1/storageclasses/csi-gce-pd
  uid: 3ad6eedf-d473-11e9-9d63-42010a800067
parameters:
  type: pd-standard
provisioner: pd.csi.storage.gke.io
reclaimPolicy: Delete
volumeBindingMode: Immediate
```

- Apply the yamls present in ./deploy/kubernetes folder

```bash
kubectl apply -f deploy/kubernetes/namespace.yaml
kubectl apply -f deploy/kubernetes/rbac.yaml
kubectl apply -f deploy/kubernetes/storage_crd.yaml
kubectl apply -f deploy/kubernetes/deployment.yaml
```

- Apply a storage

```bash
kubectl apply -f example/gke/storage.yaml
```
```yaml
# kubectl get stor magic-stor -oyaml
apiVersion: ddp.mayadata.io/v1alpha1
kind: Storage
metadata:
  annotations:
    kubectl.kubernetes.io/last-applied-configuration: |
      {"apiVersion":"ddp.mayadata.io/v1alpha1","kind":"Storage","metadata":{"annotations":{"storageprovisioner.ddp.mayadata.io/csi-attacher-name":"pd.csi.storage.gke.io","storageprovisioner.ddp.mayadata.io/storageclass-name":"csi-gce-pd"},"name":"magic-stor","namespace":"default"},"spec":{"capacity":"4Gi","nodeName":"gke-amitd-ddp-default-pool-d5aa3f95-t8p1"}}
    storageprovisioner.ddp.mayadata.io/csi-attacher-name: pd.csi.storage.gke.io
    storageprovisioner.ddp.mayadata.io/storageclass-name: csi-gce-pd
  creationTimestamp: "2019-09-19T08:15:00Z"
  generation: 1
  name: magic-stor
  namespace: default
  resourceVersion: "2375492"
  selfLink: /apis/ddp.mayadata.io/v1alpha1/namespaces/default/storages/magic-stor
  uid: 92e9d469-dab5-11e9-9d63-42010a800067
spec:
  capacity: 4Gi
  nodeName: gke-amitd-ddp-default-pool-d5aa3f95-t8p1
```

- Verify if PVC gets created with below checks
  - It has Storage as its owner
  - It gets bound to a PV via CSI dynamic provisioner

```yaml
# kubectl get pvc magic-storfrvpl -oyaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  annotations:
    pv.kubernetes.io/bind-completed: "yes"
    pv.kubernetes.io/bound-by-controller: "yes"
    storageprovisioner.ddp.mayadata.io/csi-attacher-name: pd.csi.storage.gke.io
    storageprovisioner.ddp.mayadata.io/node-name: gke-amitd-ddp-default-pool-d5aa3f95-t8p1
    volume.beta.kubernetes.io/storage-provisioner: pd.csi.storage.gke.io
  creationTimestamp: "2019-09-19T08:17:14Z"
  finalizers:
  - kubernetes.io/pvc-protection
  generateName: magic-stor
  name: magic-storfrvpl
  namespace: default
  ownerReferences:
  - apiVersion: ddp.mayadata.io/v1alpha1
    blockOwnerDeletion: true
    controller: true
    kind: Storage
    name: magic-stor
    uid: 92e9d469-dab5-11e9-9d63-42010a800067
  resourceVersion: "2375988"
  selfLink: /api/v1/namespaces/default/persistentvolumeclaims/magic-storfrvpl
  uid: e32d7590-dab5-11e9-9d63-42010a800067
spec:
  accessModes:
  - ReadWriteOnce
  dataSource: null
  resources:
    requests:
      storage: 4Gi
  storageClassName: csi-gce-pd
  volumeMode: Filesystem
  volumeName: pvc-e32d7590-dab5-11e9-9d63-42010a800067
status:
  accessModes:
  - ReadWriteOnce
  capacity:
    storage: 4Gi
  phase: Bound
```

- Verify the PV

```yaml
# kubectl get pv -oyaml
apiVersion: v1
kind: PersistentVolume
metadata:
  annotations:
    pv.kubernetes.io/provisioned-by: pd.csi.storage.gke.io
  creationTimestamp: "2019-09-19T08:17:18Z"
  finalizers:
  - kubernetes.io/pv-protection
  - external-attacher/pd-csi-storage-gke-io
  name: pvc-e32d7590-dab5-11e9-9d63-42010a800067
  resourceVersion: "2375990"
  selfLink: /api/v1/persistentvolumes/pvc-e32d7590-dab5-11e9-9d63-42010a800067
  uid: e5a1ec98-dab5-11e9-9d63-42010a800067
spec:
  accessModes:
  - ReadWriteOnce
  capacity:
    storage: 4Gi
  claimRef:
    apiVersion: v1
    kind: PersistentVolumeClaim
    name: magic-storfrvpl
    namespace: default
    resourceVersion: "2375966"
    uid: e32d7590-dab5-11e9-9d63-42010a800067
  csi:
    driver: pd.csi.storage.gke.io
    fsType: ext4
    volumeAttributes:
      storage.kubernetes.io/csiProvisionerIdentity: 1568192432358-8081-
    volumeHandle: projects/strong-eon-153112/zones/us-central1-a/disks/pvc-e32d7590-dab5-11e9-9d63-42010a800067
  persistentVolumeReclaimPolicy: Delete
  storageClassName: csi-gce-pd
  volumeMode: Filesystem
status:
  phase: Bound
```

- Verify VolumeAttachment
  - Check PVC as the owner reference

```yaml
apiVersion: storage.k8s.io/v1
kind: VolumeAttachment
metadata:
  annotations:
    csi.alpha.kubernetes.io/node-id: projects/strong-eon-153112/zones/us-central1-a/instances/gke-amitd-ddp-default-pool-d5aa3f95-t8p1
  creationTimestamp: "2019-09-19T08:17:18Z"
  finalizers:
  - external-attacher/pd-csi-storage-gke-io
  name: magic-storfrvpl
  ownerReferences:
  - apiVersion: v1
    blockOwnerDeletion: true
    controller: true
    kind: PersistentVolumeClaim
    name: magic-storfrvpl
    uid: e32d7590-dab5-11e9-9d63-42010a800067
  resourceVersion: "2376033"
  selfLink: /apis/storage.k8s.io/v1/volumeattachments/magic-storfrvpl
  uid: e5a68b43-dab5-11e9-9d63-42010a800067
spec:
  attacher: pd.csi.storage.gke.io
  nodeName: gke-amitd-ddp-default-pool-d5aa3f95-t8p1
  source:
    persistentVolumeName: pvc-e32d7590-dab5-11e9-9d63-42010a800067
status:
  attached: true
```

- Verify following after login to the respective Kubernetes node

```bash
gke-amitd-ddp-default-pool-d5aa3f95-t8p1 /home/amit_das # lsblk
NAME    MAJ:MIN RM  SIZE RO TYPE MOUNTPOINT
sda       8:0    0  100G  0 disk 
|-sda1    8:1      95.9G  0 part /mnt/stateful_partition
|-sda2    8:2    0   16M  0 part 
|-sda3    8:3    0    2G  0 part 
|-sda4    8:4    0   16M  0 part 
|-sda5    8:5    0    2G  0 part 
|-sda6    8:6       512B  0 part 
|-sda7    8:7    0  512B  0 part 
|-sda8    8:8        16M  0 part /usr/share/oem
|-sda9    8:9    0  512B  0 part 
|-sda10   8:10   0  512B  0 part 
|-sda11   8:11        8M  0 part 
`-sda12   8:12   0   32M  0 part 
sdb       8:16   0    4G  0 disk 
```

```bash
gke-amitd-ddp-default-pool-d5aa3f95-t8p1 /home/amit_das # ls /dev/disk/by-id/
google-persistent-disk-0
google-persistent-disk-0-part1
google-persistent-disk-0-part10
google-persistent-disk-0-part11
google-persistent-disk-0-part12
google-persistent-disk-0-part2
google-persistent-disk-0-part3
google-persistent-disk-0-part4
google-persistent-disk-0-part5
google-persistent-disk-0-part6
google-persistent-disk-0-part7
google-persistent-disk-0-part8
google-persistent-disk-0-part9
google-pvc-e32d7590-dab5-11e9-9d63-42010a800067
scsi-0Google_PersistentDisk_persistent-disk-0
scsi-0Google_PersistentDisk_persistent-disk-0-part1
scsi-0Google_PersistentDisk_persistent-disk-0-part10
scsi-0Google_PersistentDisk_persistent-disk-0-part11
scsi-0Google_PersistentDisk_persistent-disk-0-part12
scsi-0Google_PersistentDisk_persistent-disk-0-part2
scsi-0Google_PersistentDisk_persistent-disk-0-part3
scsi-0Google_PersistentDisk_persistent-disk-0-part4
scsi-0Google_PersistentDisk_persistent-disk-0-part5
scsi-0Google_PersistentDisk_persistent-disk-0-part6
scsi-0Google_PersistentDisk_persistent-disk-0-part7
scsi-0Google_PersistentDisk_persistent-disk-0-part8
scsi-0Google_PersistentDisk_persistent-disk-0-part9
scsi-0Google_PersistentDisk_pvc-e32d7590-dab5-11e9-9d63-42010a800067
```

### Tests
- When a Storage object is deleted
  - Assert - PVC gets deleted
  - Assert - PV gets deleted
  - Assert - VolumeAttachment gets deleted
- For a given Storage, delete its VolumeAttachment
  - Assert - VA should not get deleted due to finalizer
- For a given Storage, delete its PVC
  - Assert - PVC should not get deleted due to finalizer
- Storage Resize
- More than one Storage objects
- Storage nodename is changed
