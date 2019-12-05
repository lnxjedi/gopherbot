# Configuration File Loading

**Gopherbot** uses **YAML** for it's configuration files, and Go text templates for expanding the files it reads. Any time **Gopherbot** loads a configuration file, say `conf/gopherbot.yaml`, it first looks for the file in the installation directory, and loads and expands that file if found. Next it looks for the same file in the custom configuration directory; if found, it loads and expands that file, then recursively merges the two data structures:
* Map values merge and override
* Array values are replaced

To illustrate with an example, take the following two excerpts from `gopherbot.yaml`:

Default from the install archive:
```yaml
ProtocolConfig:
  StartChannel: general
  Channels:
  - random
  - general
  StartUser: alice
```

Custom local configuration:
```yaml
ProtocolConfig:
  StartChannel: jobs
  Channels:
  - jobs
  - general
```

The resulting configuration would be:
```yaml
ProtocolConfig:
  StartChannel: jobs
  Channels:
  - jobs
  - general
  StartUser: alice
```

## Template Expansion

**Gopherbot** uses standard [Go text templates](https://golang.org/pkg/text/template) for expanding the configuration files it reads. In addition to the stock syntactic elements, the following functions and methods are available:

### The `env` function

```
{{ env "GOPHER_ALIAS" }}
```

This would expand to the value of the `GOPHER_ALIAS` environment variable.

### The `default` function

```
{{ env "GOPHER_ALIAS" | default ";" }}
```

The `default` function takes two arguments, and returns the first argument if it's length is > 0, the second argument otherwise. The example would expand to the value of the `GOPHER_ALIAS` environment variable if set, or `;` otherwise.

## The `decrypt` function

```
ProtocolConfig:
  SlackToken: xoxb-18000000000-470000000000-{{ decrypt "xxxxx" }}
```

To make it safe to store secret values in configuration, administrators can send a direct message to the robot requesting encryption, e.g.:

```
c:(direct)/u:alice -> encrypt MyLousyPassword
(dm:alice): RPzxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx+9rf==
```

The encrypted value can then be pasted in to the `decrypt` function. See the section on *encryption* for more information.

**NOTE:** In the example, the common, low-entropy portion of a *slack* token is left as-is, and only the high-entropy portion of the token is encrypted, to prevent attacks where a portion of the encrypted value is known.

## The `.Include` method

```
{{ .Include "terminal.yaml" }}
```

Note that `.Include` is a *method* on the configuration file object, which is either an install file or a custom file. If the above example is present in the installed `conf/gopherbot.yaml`, it will include *only* the installed `conf/terminal.yaml`, if present, and ignore that file if it's also present in the custom directory.
