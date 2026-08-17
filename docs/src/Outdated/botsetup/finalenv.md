# Finish the `.env` File

First, use `tr` to dump a flattened version of the `deploy_key` private key to the `.env` file:
```
$ cat custom/ssh/deploy_key | tr ' \n' '_:' >> .env
```

Now edit the `.env` file and stick `GOPHER_DEPLOY_KEY=` in front of that mess, and add values for `GOPHER_PROTOCOL` and `GOPHER_CUSTOM_REPOSITORY`. Using **Clu** as an example, the end result should look something like this:
```
GOPHER_ENCRYPTION_KEY=<redacted>
GOPHER_PROTOCOL=slack
GOPHER_CUSTOM_REPOSITORY=git@github.com:parsley42/clu-gopherbot.git
GOPHER_DEPLOY_KEY=-----BEGIN_OPENSSH_PRIVATE_KEY-----:lksjflsjdf<muchjunkremoved>ljsdflsjdf:-----END_OPENSSH_PRIVATE_KEY-----:
```

> Note that `GOPHER_DEPLOY_KEY` is a very long line, a bit over 2k. The format with spaces and newlines replaced was chosen as most compatible with the shell, docker, and `.env`-reading libraries.
