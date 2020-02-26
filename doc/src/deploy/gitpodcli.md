# Using Gitpod

[Gitpod](https://gitpod.io) is a pretty good way to work with your robot's CLI, with the caveat that your robot needs to be stored in a public repository to use the free service. The standard robot includes a `.gitpod.yml` file, so if you've already signed up for **Gitpod** you can just visit: https://gitpod.io/#https://github.com/\<org\>/\<robot-repo\>, or click the **Gitpod** button if you've installed the [browser extension](https://www.gitpod.io/docs/browser-extension).

When your workspace opens, you can use the `File` menu to create a new `.env` file directly in the `/workspace` folder. Once you've pasted in the contents and saved the file, just start the robot in the lower terminal pane:
```
gitpod /workspace $ gopherbot 
2020/02/20 19:52:01 Debug: Checking os.Stat for dir 'custom' from wd '': stat custom: no such file or directory
2020/02/20 19:52:01 Debug: Checking os.Stat for dir 'conf' from wd '': stat conf: no such file or directory
2020/02/20 19:52:01 Initialized logging ...
...
2020/02/20 19:52:02 Info: Robot is initialized and running
2020/02/20 19:52:02 Info: Creating bootstrap pipeline for git@github.com:parsley42/clu-gopherbot.git
2020/02/20 19:52:02 Warning: /workspace/custom/git/config not found, git push will fail
2020/02/20 19:52:02 Info: ssh-init starting in bootstrap mode
2020/02/20 19:52:02 Warning: Output from stderr of external command '/opt/gopherbot/tasks/ssh-init.sh': Identity added: (stdin) (parse@hakuin.localdomain)
2020/02/20 19:52:03 Warning: Output from stderr of external command '/opt/gopherbot/tasks/git-clone.sh': Cloning into '.'...
2020/02/20 19:52:03 Warning: Repository pipeline not found in job  (wd: /workspace/custom, repo: not set), ignoring
2020/02/20 19:52:03 Info: Restart triggered in pipeline 'bootstrap' with 0 pipelines running (including this one)
2020/02/20 19:52:03 Info: Stop called with 1 pipelines running
2020/02/20 19:52:03 Info: Restarting...
2020/02/20 19:52:03 Info: Stopping signal handler
Terminal connector running; Use '|c<channel|?>' to change channel, or '|u<user|?>' to change user
OUT: GOPHER_PROTOCOL is terminal
clu-jobs: Starting restore of robot state...
OUT: Agent pid 1764
ERR: Identity added: /workspace/custom/ssh/manage_rsa (parse@hakuin.localdomain)
ERR: Cloning into '.'...
clu-jobs: Restore finished
OUT: unset SSH_AUTH_SOCK;
OUT: unset SSH_AGENT_PID;
OUT: echo Agent pid 1764 killed;
c:general/u:alice -> <ctrl-d>
general: @alice Later gator!
Terminal connector finished
gitpod /workspace $ gopherbot list
links:links
pythondemo:memory
lists:listmap
...
```
