# Gopherbot Helm Chart

This helm chart can be used to deploy your **Gopherbot** robot to your Kubernetes cluster. This README was written for helm3.

Before using this chart you should have already created a **Gopherbot** robot with an associated git repository, along with a `.env` file for bootstrapping your robot.

Most of the data in values.yaml should come directly from the `.env` file. The additional `name` value (default: "robot") gives you the ability to deploy multiple robots to your cluster.

As an example, for a simple deployment of my "Clu" robot, I can use a `clu-values.yaml` file like this:
```yaml
# defaults to robot, set if you want multiple robots
robotName: "clu"

# These values should come from the .env file created during setup
# clone URL for the repository using ssh; GOPHER_CUSTOM_REPOSITORY
robotRepository: "git@github.com:parsley42/clu-gopherbot.git"
# trivially encoded read-only deploy key; GOPHER_DEPLOY_KEY
deployKey: "<redacted - use value from .env file>"
# secret used for encryption/decryption; GOPHER_ENCRYPTION_KEY
encryptionKey: "<redacted - use value from .env file>"
# protocol for connecting to team chat; GOPHER_PROTOCOL
protocol: slack
```

Then, to deploy Clu to my cluster:
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
