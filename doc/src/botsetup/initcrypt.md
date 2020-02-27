# Initialize Encryption
Now that you've set the robot's encryption key, you can initialize encryption and create the binary key (`binary-encrypted-key`) encrypted with your `GOPHER_ENCRYPTION_KEY`. Start the default robot using the terminal connector, and the binary encryption key will be automatically created in `custom/binary-encrypted-key`; use `ctrl-d` to exit:

```
davidparsley@penguin:/home/robots/clu$ ./gopherbot 
2020/02/19 15:29:57 Debug: Checking os.Stat for dir 'custom' from wd '': stat custom: no such file or directory
2020/02/19 15:29:57 Debug: Checking os.Stat for dir 'conf' from wd '': stat conf: no such file or directory
2020/02/19 15:29:57 Warning: Starting unconfigured; no robot.yaml/gopherbot.yaml found
Terminal connector running; Use '|c<channel|?>' to change channel, or '|u<user|?>' to change user
c:general/u:alice -> 2020/02/19 15:29:57 Warning: GOPHER_CUSTOM_REPOSITORY not set, not bootstrapping
general: *******
general: Type ';setup' to continue setup...
c:general/u:alice -> <ctrl-d>
general: @alice Adios
Terminal connector finished
davidparsley@penguin:/home/robots/clu$ ls custom/
binary-encrypted-key
```
