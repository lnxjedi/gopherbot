## Gopherbot Resources

* `userdaemon.te` - update for SELinux to allow Gopherbot to run as a normal user; see comments in file
* `gopherbot.service` - sample Gopherbot unit file for systemd
* `terminal.sh` - convenience script for starting a robot with the terminal connector
* `docker/` - Dockerfiles and Makefiles for creating Gopherbot Docker Images
   * `docker/Makefile` - Makefile with examples for starting dev and prod containers
   * `docker/amazon/` / `docker/ubuntu/` - Makefiles and Dockerfiles for Amazon Linux and Ubuntu based images
