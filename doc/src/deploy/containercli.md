# Container Operation

The next best option to a local install is running the CLI in a container; just drop your robot's `.env` file in a directory and run the **gitpod** image. Using `podman` as an example:
```
[parse@hakuin ~]$ mkdir clu
[parse@hakuin ~]$ cd clu/
[parse@hakuin clu]$ vim .env
[parse@hakuin clu]$ podman run -it --env-file .env lnxjedi/gopherbot:gitpod
root@061053d1e395:/home# gopherbot 
2020/02/20 19:23:32 Debug: Checking os.Stat for dir 'custom' from wd '': stat custom: no such file or directory
2020/02/20 19:23:32 Debug: Checking os.Stat for dir 'conf' from wd '': stat conf: no such file or directory
2020/02/20 19:23:32 Initialized logging ...
...
2020/02/20 19:26:29 Info: Robot is initialized and running
2020/02/20 19:26:29 Info: Creating bootstrap pipeline for git@github.com:parsley42/clu-gopherbot.git
2020/02/20 19:26:29 Warning: /home/custom/git/config not found, git push will fail
2020/02/20 19:26:29 Info: ssh-init starting in bootstrap mode
2020/02/20 19:26:29 Warning: Output from stderr of external command '/opt/gopherbot/tasks/ssh-init.sh': Identity added: (stdin) (parse@hakuin.localdomain)
2020/02/20 19:26:30 Warning: Output from stderr of external command '/opt/gopherbot/tasks/git-clone.sh': Cloning into '.'...
2020/02/20 19:26:30 Warning: Repository pipeline not found in job  (wd: /home/custom, repo: not set), ignoring
2020/02/20 19:26:30 Info: Restart triggered in pipeline 'bootstrap' with 0 pipelines running (including this one)
2020/02/20 19:26:30 Info: Stop called with 1 pipelines running
2020/02/20 19:26:30 Info: Restarting...
2020/02/20 19:26:30 Info: Stopping signal handler
Terminal connector running; Use '|c<channel|?>' to change channel, or '|u<user|?>' to change user
OUT: GOPHER_PROTOCOL is terminal
clu-jobs: Starting restore of robot state...
OUT: Agent pid 440
ERR: Identity added: /home/custom/ssh/manage_rsa (parse@hakuin.localdomain)
ERR: Cloning into '.'...
clu-jobs: Restore finished
OUT: unset SSH_AUTH_SOCK;
OUT: unset SSH_AGENT_PID;
OUT: echo Agent pid 440 killed;
c:general/u:alice -> <ctrl-d>
general: @alice Sayonara!
Terminal connector finished
root@7c0df37ca541:/home# gopherbot list
pythondemo:memory
links:links
lists:listmap
...
...
