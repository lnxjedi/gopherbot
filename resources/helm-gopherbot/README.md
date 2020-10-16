# Gopherbot Helm Chart

This helm chart can be used to deploy your **Gopherbot** robot to your Kubernetes cluster. This README was written for helm3.

Before using this chart you should have already created a **Gopherbot** robot with an associated git repository, along with a `.env` file for bootstrapping your robot.

You can deploy your robot with this helm chart with two basic steps:
1. Store your robot's secrets in the target namespace using the `.env`
2. Deploy the robot with a helm values file

The values file should have no sensitive data, and could be stored in source control management.

## Generating and Storing Your Robot's Secrets

The first step is to store your robot's sensitive data in a secret in the robot's target namespace. You can use the included `generate-secret.sh` script, like so:
```
$ ./resources/helm-gopherbot/generate-secret.sh robot-secrets ~/robot/.env | kubectl -n robot apply -f -
secret/robot-secrets created
```

## Creating your Robot's Values File

The second step is configuring your robot. Make a copy of `resources/helm-gopherbot/values.yaml` and edit it. Most items can be removed, but you may wish to set a value for the `robotDataVolume` if you want your robot to have a persistent data volume. Since `gopherbot` runs non-root, you should probably also set `fsGroup` in the `podSecurityContext`, so the mount will be writeable.

For example:
```yaml
robotDataVolume:
  persistentVolumeClaim:
    claimName: robot-pvc

podSecurityContext:
  fsGroup: 1 # "daemon"
```

If you're going to have multiple robots in a single namespace, you should override the values for secrets and fullname; using `clu` as an example:
```yaml
robotDataVolume:
  persistentVolumeClaim:
    claimName: clu-pvc

robotSecrets: clu-secrets

fullnameOverride: clu-gopherbot
```

## Deploying Your Robot with Helm

Then, to deploy your robot to your cluster (using `clu` as an example):
```
$ helm install clu ./helm-gopherbot --values=clu-values.yaml
NAME: clu
LAST DEPLOYED: Tue Jun  2 14:27:55 2020
NAMESPACE: default
STATUS: deployed
REVISION: 1
TEST SUITE: None
[parse@joshu clu]$ k get deployments.apps 
NAME             READY   UP-TO-DATE   AVAILABLE   AGE
clu-gopherbot    0/1     1            0           10s
...
```
