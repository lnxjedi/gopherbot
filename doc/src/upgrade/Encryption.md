## Early Encryption Initialization

**Gopherbot** uses two separate keys for encrypting and decrypting secret data:
* The start-up key, normally 32 printable `ASCII` characters, used only for decrypting the binary, 32-byte random encryption key
* The permanent runtime encryption key, 32 random bytes generated on the first run 

In version 1 and early snapshots of version 2, the runtime key was stored in the brain, and encryption was only initialized after the robot's brain provider and protocol connector. Secrets for these had to be provided in the environment, or unencrypted in `gopherbot.yaml`. Version 2 addresses this by storing the binary encrypted runtime key in a `base64` encoded file, `custom/binary-encrypted-key`. This allows storing encrypted values for e.g. the *slack token* or *AWS credentials* in `gopherbot.yaml`, and only needing to provide the start-up `GOPHER_ENCRYPTION_KEY` in the environment, or in an external environment file - `$GOPHER_HOME/.env` or `$GOPHER_HOME/private/environment`.

For a robot instance with a file-backed brain, the following command can be used to generate the version 2 key file:

```bash
$ cat /path/to/brain/bot:encryptionKey | base64 -w 0 > /path/to/custom/binary-encrypted-key
```

This file can then be committed to the custom configuration repository.

## Default Brain Encryption

In version 2, **Gopherbot** defaults to an encrypted brain, which requires a `GOPHER_ENCRYPTION_KEY` to be set. To run your robot without brain encryption, you need to explicitly turn it off in your custom `gopherbot.yaml`:

```yaml
EncryptBrain: false
```
