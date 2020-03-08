# Local Install

If you've [installed Gopherbot](../install/ManualInstall.md) on a Linux host, you can just create an empty directory, add your `.env` file, and start **gopherbot** in terminal mode - letting the **bootstrap** plugin retrieve the rest of your robot:
```
[parse@hakuin ~]$ mkdir clu
[parse@hakuin ~]$ cd clu/
[parse@hakuin clu]$ vim .env
[parse@hakuin clu]$ ln -s /opt/gopherbot/gopherbot .
[parse@hakuin clu]$ ./gopherbot 
2020/02/20 14:14:20 Debug: Checking os.Stat for dir 'custom' from wd '': stat custom: no such file or directory
2020/02/20 14:14:20 Debug: Checking os.Stat for dir 'conf' from wd '': stat conf: no such file or directory
2020/02/20 14:14:20 Initialized logging ...
...
2020/02/20 14:14:21 Info: Robot is initialized and running
2020/02/20 14:14:21 Info: Creating bootstrap pipeline for git@github.com:parsley42/clu-gopherbot.git
2020/02/20 14:14:21 Warning: /home/parse/clu/custom/git/config not found, git push will fail
2020/02/20 14:14:21 Info: ssh-init starting in bootstrap mode
2020/02/20 14:14:21 Warning: Output from stderr of external command '/home/gopherbot/tasks/ssh-init.sh': Identity added: (stdin) (parse@hakuin.localdomain)
2020/02/20 14:14:22 Warning: Output from stderr of external command '/home/gopherbot/tasks/git-clone.sh': Cloning into '.'...
2020/02/20 14:14:22 Warning: Repository pipeline not found in job  (wd: /home/parse/clu/custom, repo: not set), ignoring
2020/02/20 14:14:22 Info: Restart triggered in pipeline 'bootstrap' with 0 pipelines running (including this one)
2020/02/20 14:14:22 Info: Stop called with 1 pipelines running
2020/02/20 14:14:22 Info: Restarting...
2020/02/20 14:14:22 Info: Stopping signal handler
Terminal connector running; Use '|c<channel|?>' to change channel, or '|u<user|?>' to change user
clu-jobs: Starting restore of robot state...
OUT: Agent pid 11337
ERR: Identity added: /home/parse/clu/custom/ssh/manage_key (parse@hakuin.localdomain)
ERR: Cloning into '.'...
ERR: Warning: Permanently added the RSA host key for IP address '192.30.253.112' to the list of known hosts.
clu-jobs: Restore finished
OUT: unset SSH_AUTH_SOCK;
OUT: unset SSH_AGENT_PID;
OUT: echo Agent pid 11337 killed;
c:general/u:alice -> <ctrl-d>
general: @alice Adios
Terminal connector finished
[parse@hakuin clu]$ gopherbot list
pythondemo:memory
links:links
lists:listmap
...
```
