### Steps
Go through following steps to make storage-provisioner work in EKS.

- Get a EKS cluster with Kubernetes 1.14

```bash
  storage-provisioner > kubectl get node
NAME                                           STATUS   ROLES    AGE   VERSION
ip-192-168-44-176.us-east-2.compute.internal   Ready    <none>   82m   v1.14.6-eks-5047ed
ip-192-168-9-76.us-east-2.compute.internal     Ready    <none>   82m   v1.14.6-eks-5047ed
ip-192-168-90-90.us-east-2.compute.internal    Ready    <none>   82m   v1.14.6-eks-5047ed
```

- git clone below project with appropriate branch (i.e. in this case 0.4.0)
  - https://github.com/kubernetes-sigs/aws-ebs-csi-driver/blob/release-0.4.0/

```bash
  aws-ebs-csi-driver > git status
On branch release-0.4.0
Your branch is up to date with 'origin/release-0.4.0'.

Changes not staged for commit:
  (use "git add <file>..." to update what will be committed)
  (use "git checkout -- <file>..." to discard changes in working directory)

	modified:   deploy/kubernetes/secret.yaml

```

```bash
  aws-ebs-csi-driver > kubectl apply -f deploy/kubernetes/secret.yaml 
secret/aws-secret configured

 aws-ebs-csi-driver > kubectl get secret -n kube-system | grep aws
aws-cloud-provider-token-gzsrz                   kubernetes.io/service-account-token   3      66m
aws-node-token-tbmtm                             kubernetes.io/service-account-token   3      66m
aws-secret                                       Opaque                                2      22m
  aws-ebs-csi-driver > 

```

```bash
  aws-ebs-csi-driver > kubectl get sts -n kube-system
NAME                 READY   AGE
ebs-csi-controller   1/1     4m36s
  aws-ebs-csi-driver > 
```

```bash
  aws-ebs-csi-driver > kubectl get daemonset -n kube-system
NAME           DESIRED   CURRENT   READY   UP-TO-DATE   AVAILABLE   NODE SELECTOR                 AGE
aws-node       3         3         3       3            3           <none>                        67m
ebs-csi-node   3         3         3       3            3           beta.kubernetes.io/os=linux   4m42s
kube-proxy     3         3         3       3            3           <none>                        67m
  aws-ebs-csi-driver > 
```

```bash
  aws-ebs-csi-driver > kubectl get po -n kube-system | grep csi
ebs-csi-controller-0       4/4     Running   0          5m52s
ebs-csi-node-4px7g         3/3     Running   0          5m51s
ebs-csi-node-74mbt         3/3     Running   0          5m51s
ebs-csi-node-jj9nk         3/3     Running   0          5m51s
```

```bash
  aws-ebs-csi-driver > kubectl get csinode
NAME                                           CREATED AT
ip-192-168-44-176.us-east-2.compute.internal   2019-09-25T08:47:56Z
ip-192-168-9-76.us-east-2.compute.internal     2019-09-25T08:47:55Z
ip-192-168-90-90.us-east-2.compute.internal    2019-09-25T08:47:56Z
```

- Following steps are from this project
  - i.e. https://github.com/AmitKumarDas/storage-provisioner

```bash
kubectl apply -f deploy/kubernetes/namespace.yaml

kubectl apply -f deploy/kubernetes/rbac.yaml

kubectl apply -f deploy/kubernetes/storage_crd.yaml

kubectl apply -f deploy/kubernetes/deployment.yaml
```

```bash
  storage-provisioner > kubectl get pod -n ddp
NAME                                       READY   STATUS    RESTARTS   AGE
ddp-storage-provisioner-6fc7d9dcc4-nnrnv   1/1     Running   0          5m30s
```

```bash
kubectl apply -f example/eks/sc.yaml

kubectl apply -f example/eks/storage.yaml
```

```bash
  storage-provisioner > kubectl get stor
NAME             CAPACITY   NODENAME                                       STATUS
magic-aws-stor   3Gi        ip-192-168-44-176.us-east-2.compute.internal   
```

```bash
  storage-provisioner > kubectl get pvc
NAME                           STATUS   VOLUME                                     CAPACITY   ACCESS MODES   STORAGECLASS   AGE
default-magic-aws-stor-hkrqk   Bound    pvc-0cbd7668-df75-11e9-bd75-0a35aab5d502   3Gi        RWO            ebs-sc         10s

```bash
  storage-provisioner > kubectl get volumeattachment
NAME                           CREATED AT
default-magic-aws-stor-hkrqk   2019-09-25T09:15:51Z

```

```bash
 storage-provisioner > kubectl get pvc -oyaml
apiVersion: v1
items:
- apiVersion: v1
  kind: PersistentVolumeClaim
  metadata:
    annotations:
      pv.kubernetes.io/bind-completed: "yes"
      pv.kubernetes.io/bound-by-controller: "yes"
      storageprovisioner.ddp.mayadata.io/csi-attacher-name: ebs.csi.aws.com
      storageprovisioner.ddp.mayadata.io/node-name: ip-192-168-9-76.us-east-2.compute.internal
      volume.beta.kubernetes.io/storage-provisioner: ebs.csi.aws.com
    creationTimestamp: "2019-09-25T09:38:43Z"
    finalizers:
    - kubernetes.io/pvc-protection
    generateName: default-magic-aws-stor-
    name: default-magic-aws-stor-sgzld
    namespace: default
    ownerReferences:
    - apiVersion: ddp.mayadata.io/v1alpha1
      blockOwnerDeletion: true
      controller: true
      kind: Storage
      name: magic-aws-stor
      uid: 438180a9-df78-11e9-bd75-0a35aab5d502
    resourceVersion: "11198"
    selfLink: /api/v1/namespaces/default/persistentvolumeclaims/default-magic-aws-stor-sgzld
    uid: 4383c0bd-df78-11e9-bd75-0a35aab5d502
  spec:
    accessModes:
    - ReadWriteOnce
    resources:
      requests:
        storage: 3Gi
    storageClassName: ebs-sc
    volumeMode: Filesystem
    volumeName: pvc-4383c0bd-df78-11e9-bd75-0a35aab5d502
  status:
    accessModes:
    - ReadWriteOnce
    capacity:
      storage: 3Gi
    phase: Bound
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```

```bash
  storage-provisioner > kubectl get pv -oyaml
apiVersion: v1
items:
- apiVersion: v1
  kind: PersistentVolume
  metadata:
    annotations:
      pv.kubernetes.io/provisioned-by: ebs.csi.aws.com
    creationTimestamp: "2019-09-25T09:38:51Z"
    finalizers:
    - kubernetes.io/pv-protection
    - external-attacher/ebs-csi-aws-com
    name: pvc-4383c0bd-df78-11e9-bd75-0a35aab5d502
    resourceVersion: "11200"
    selfLink: /api/v1/persistentvolumes/pvc-4383c0bd-df78-11e9-bd75-0a35aab5d502
    uid: 487b8d6b-df78-11e9-aec7-066bff9ae944
  spec:
    accessModes:
    - ReadWriteOnce
    capacity:
      storage: 3Gi
    claimRef:
      apiVersion: v1
      kind: PersistentVolumeClaim
      name: default-magic-aws-stor-sgzld
      namespace: default
      resourceVersion: "11180"
      uid: 4383c0bd-df78-11e9-bd75-0a35aab5d502
    csi:
      driver: ebs.csi.aws.com
      fsType: ext4
      volumeAttributes:
        storage.kubernetes.io/csiProvisionerIdentity: 1569401276608-8081-ebs.csi.aws.com
      volumeHandle: vol-03498a2445f718b3b
    nodeAffinity:
      required:
        nodeSelectorTerms:
        - matchExpressions:
          - key: topology.ebs.csi.aws.com/zone
            operator: In
            values:
            - us-east-2c
    persistentVolumeReclaimPolicy: Delete
    storageClassName: ebs-sc
    volumeMode: Filesystem
  status:
    phase: Bound
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```

```bash
 storage-provisioner > kubectl get volumeattachment -oyaml
apiVersion: v1
items:
- apiVersion: storage.k8s.io/v1
  kind: VolumeAttachment
  metadata:
    annotations:
      csi.alpha.kubernetes.io/node-id: i-0e3a59d520a62416a
    creationTimestamp: "2019-09-25T09:38:51Z"
    finalizers:
    - external-attacher/ebs-csi-aws-com
    name: default-magic-aws-stor-sgzld
    ownerReferences:
    - apiVersion: v1
      blockOwnerDeletion: true
      controller: true
      kind: PersistentVolumeClaim
      name: default-magic-aws-stor-sgzld
      uid: 4383c0bd-df78-11e9-bd75-0a35aab5d502
    resourceVersion: "11205"
    selfLink: /apis/storage.k8s.io/v1/volumeattachments/default-magic-aws-stor-sgzld
    uid: 487ea18c-df78-11e9-bd75-0a35aab5d502
  spec:
    attacher: ebs.csi.aws.com
    nodeName: ip-192-168-9-76.us-east-2.compute.internal
    source:
      persistentVolumeName: pvc-4383c0bd-df78-11e9-bd75-0a35aab5d502
  status:
    attached: true
    attachmentMetadata:
      devicePath: /dev/xvdba
kind: List
metadata:
  resourceVersion: ""
  selfLink: ""
```
