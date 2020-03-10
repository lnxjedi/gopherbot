# Initialize Encryption
Now that you've set the robot's encryption key, you can initialize encryption and create the binary key (`binary-encrypted-key`) encrypted with your `GOPHER_ENCRYPTION_KEY`. Start the default robot using the terminal connector, and the binary encryption key will be automatically created in `custom/binary-encrypted-key`; use `ctrl-d` to exit:

```
davidparsley@penguin:/home/robots/clu$ ./gopherbot 
2020/03/09 18:29:43 Logging to robot.log; warnings and errors duplicated to stdout
Terminal connector running; Use '|c<channel|?>' to change channel, or '|u<user|?>' to change user
general: *******
general: Welcome to the *Gopherbot* terminal connector. Since no configuration was detected, you're
connected to 'floyd', the default robot.
general: If you've started the robot by mistake, just hit ctrl-D to exit and try 'gopherbot --help';
otherwise feel free to play around with the default robot - you can start by typing 'help'.
c:general/u:alice -> exit
general: @alice Hasta la vista!
Terminal connector finished
davidparsley@penguin:/home/robots/clu$ ls custom/
binary-encrypted-key
```
