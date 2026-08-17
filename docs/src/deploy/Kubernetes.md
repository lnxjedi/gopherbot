# Kubernetes Example

Kubernetes is a good fit when:

- your team already deploys operational tooling in-cluster
- you want secrets and restart behavior managed by the platform
- the robot's external dependencies fit cleanly into a container image

The repository ships an example manifest at `resources/deploy-gopherbot.yaml` and Helm material under `resources/helm-gopherbot/`.

## Minimum checklist

- store the robot's environment values as a Kubernetes secret
- mount or inject those values into the container
- run only one production instance for a given robot unless you have planned for coordination
- ship any extra system dependencies in the image instead of assuming they exist in the cluster
