# Create the GOPHER_HOME directory
Similar to an [Ansible](https://www.ansible.com/) playbook, a **Gopherbot** robot is heavily oriented around a standard directory structure for a given robot. To begin with, create an empty directory for your robot; `/var/lib/robots` or `/home/robots` are good places for this. I normally name the directory after the robot; in this example setup, we'll use `clu`:
```shell
davidparsley@penguin:/home/robots$ mkdir clu
davidparsley@penguin:/home/robots$ cd clu/
davidparsley@penguin:/home/robots/clu$ ln -s /opt/gopherbot/gopherbot .
```

> Note that we've created a symlink to the **Gopherbot** binary for convenience, and this is required for the rest of this chapter.
