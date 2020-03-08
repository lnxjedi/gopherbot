# Generate SSH Keypairs
Now generate the ssh keypairs for your robot - use the passphrase you encrypted previously for generating both `manage_key` and `robot_key`; finally create an unencrypted `deploy_key` and move it out of the repo:
```shell
davidparsley@penguin:/home/robots/clu$ mkdir custom/ssh
davidparsley@penguin:/home/robots/clu$ ssh-keygen -N "VeryLovelyNicePassword" -C "clu@linuxjedi.org" -f custom/ssh/manage_key
Generating public/private rsa key pair.
Your identification has been saved in custom/ssh/manage_key.
Your public key has been saved in custom/ssh/manage_key.pub.
The key fingerprint is:
...
davidparsley@penguin:/home/robots/clu$ ssh-keygen -N "VeryLovelyNicePassword" -C "clu@linuxjedi.org" -f custom/ssh/robot_key
Generating public/private rsa key pair.
Your identification has been saved in custom/ssh/robot_key.
Your public key has been saved in custom/ssh/robot_key.pub.
The key fingerprint is:
...
davidparsley@penguin:/home/robots/clu$ ssh-keygen -N "" -C "clu@linuxjedi.org" -f custom/ssh/deploy_key
Generating public/private rsa key pair.
Your identification has been saved in custom/ssh/deploy_key.
Your public key has been saved in custom/ssh/deploy_key.pub.
The key fingerprint is:
...
davidparsley@penguin:/home/robots/clu$ mv custom/ssh/deploy_key .
```

>  Note that deploy_key is unencrypted, and not kept in the robot's repository which is rooted in the `custom` directory.
