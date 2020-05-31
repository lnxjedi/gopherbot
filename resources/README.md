## Gopherbot Resources

* `deploy-gopherbot.yaml` - for deploying to Kubernetes with `kubectl apply ...`; see comments in file
* `robot.service` - sample Gopherbot unit file for systemd
* `user-robot.service` - sample Gopherbot unit file for running as a user systemd service (as your user)
* `userdaemon.te` - update for SELinux to allow Gopherbot to run as a normal user; see comments in file (possibly outdated)
* `docker/` - Dockerfiles and Makefiles for creating Gopherbot Docker Images
   * `docker/Makefile` - Makefile with examples for starting dev and prod containers
   * `docker/*/` - Makefiles and Dockerfiles for container images
