# Create the Initial .env File
Though it's possible to run **Gopherbot** with neither encryption nor a *git* repository, this example documents the more common scenario. To begin with, create a `.env` file in your new directory with contents similar to the following:
```shell
davidparsley@penguin:/home/robots/clu$ echo "GOPHER_ENCRYPTION_KEY=SomeLongStringAtLeast32CharsOrItWillFail" > .env
```
Note that the robot will only use the first 32 chars of the encryption key, so no need to go crazy here. Keep your `GOPHER_ENCRYPTION_KEY` in a safe place, outside of any git repository. Your robot will need it, along with `$GOPHER_HOME/custom/binary-encrypted-key` (kept in the robot's repository), to decrypt it's secrets - such as a Slack token.
