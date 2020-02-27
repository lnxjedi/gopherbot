# Generate SSH Keypairs
Now generate the ssh keypairs for your robot - use the passphrase you encrypted previously for generating both `manage_rsa` and `robot_rsa`; finally create an unencrypted `deploy_rsa` and move it out of the repo:
```shell
davidparsley@penguin:/home/robots/clu$ mkdir custom/ssh
davidparsley@penguin:/home/robots/clu$ ssh-keygen -N "VeryLovelyNicePassword" -C "clu@linuxjedi.org" -f custom/ssh/manage_rsa
Generating public/private rsa key pair.
Your identification has been saved in custom/ssh/manage_rsa.
Your public key has been saved in custom/ssh/manage_rsa.pub.
The key fingerprint is:
...
davidparsley@penguin:/home/robots/clu$ ssh-keygen -N "VeryLovelyNicePassword" -C "clu@linuxjedi.org" -f custom/ssh/robot_rsa
Generating public/private rsa key pair.
Your identification has been saved in custom/ssh/robot_rsa.
Your public key has been saved in custom/ssh/robot_rsa.pub.
The key fingerprint is:
...
davidparsley@penguin:/home/robots/clu$ ssh-keygen -N "" -C "clu@linuxjedi.org" -f custom/ssh/deploy_rsa
Generating public/private rsa key pair.
Your identification has been saved in custom/ssh/deploy_rsa.
Your public key has been saved in custom/ssh/deploy_rsa.pub.
The key fingerprint is:
...
davidparsley@penguin:/home/robots/clu$ mv custom/ssh/deploy_rsa .
```

>  Note that deploy_rsa is unencrypted, and not kept in the robot's repository which is rooted in the `custom` directory.
