Since Gopherbot is designed for ChatOps with the idea of being an 'Enterprise Sudo', it is important to discuss security-related issues. It is expected that as team chat services and therefore ChatOps becomes more prevalent in mainstream IT, understanding of ChatOps security issues will improve and mature. Laid out here are a few general considerations along with some of Gopherbot's specific security-related features.

## Plugin (non-)Separation
Gopherbot's design is intended to allow _eventual_ support for a strong separation between external plugins, so that e.g. internally developed plugins can (more) safely coexist with 3rd-party external plugins. This is not yet fully implemented, however the API design should accommodate it. Currently the robot and all external plugins run as the robot user; mainly this means that all external plugins can read whatever files the main gopherbot process can read, including the file-based brain.

### Trusted (internally-developed) and Untrusted (third party) Plugins
Gopherbot is designed with an eye towards future proliferation of third party plugins - from managing cloud provider infrastructure, to ordering pizza, to generating memes or spitting out random facts about cats and Chuck Norris. Currently there are only a small number of plugins available, but it's still important to discuss and consider these aspects of ChatOps security.

When running Gopherbot for doing real, privileged, security-sensitive work, best practices would dictate:
 * Never run both trusted and untrusted plugins in the same instance of Gopherbot
 * Never invite a bot running untrusted plugins into a channel where a bot with trusted plugins is running

The reasoning here is that plugins have the ability to listen and respond to everything said, and a user might not always be certain of what plugin they're interacting with. That being said, many plugins (such as `weather`) are very short and easy to read/audit for malicious code.

## Visibility
Each plugin can specify one or more of `Users`, `Channels`, `AllChannels`, `RequireAdmin`, `AllowDirect` and `DirectOnly` that will limit who a plugin is visible to, and whether it can be accessed in a given channel or via direct message. For instance, you could allow certain security-sensitive plugins to be visible only in a few invite-only private channels. Note that if a given plugin is available to a user only in certain channels, `help <keyword>` will list the channels where a plugin is available.

Additionally, being able to restrict based on channels means that potentially security-sensitive operations will always be performed in the view of other members of the team, adding another level of protection.

## Authorization
If a plugin is available to the user, the robot will then check authorization, if configured. Instead of creating a pluggable interface for e.g. group membership, or other authorization primitives, Gopherbot uses the notion of an "Authorizer" plugin that gets called with the command `authorize`, and these arguments:
the name of the plugin being authorized, an optional group/role name, followed by the command and any arguments to the command. The plugin can perform look-ups or optionally interact with the user, and is expected to exit with `bot.Success` if the user is authorized, `bot.Fail` if the user isn't authorized, or `bot.MechanismFail` / `bot.ConfigurationError` if e.g. LDAP or some other central service couldn't be reached or is misconfigured. Note that the libraries define constants for these values.

Authorization is useful for all kinds of cases where a given plugin may be available in several channels, but uses different resources based on the channel and simply limiting visibility isn't sufficient. It's also useful for implementing e.g. group security. The main upside is that it gives the bot administrator the ability to implement arbitrary logic for determining authorization, but that's also the main downside - it may require scripting to configure certain types of authorization.

## Elevation
Finally, if the user passes the authorization check, the robot will then check for elevation if a given command is listed in `ElevatedCommands` or `ElevateImmediateCommands`. Elevation behaves similarly to `sudo`, in that the user may be required to supply a second form of authentication (mfa / 2fa) before an action is allowed. Individual elevation plugins may be configurable with a timeout for `ElevatedCommands`, such that a user can continue to perform elevated operations for a period of time before re-authentication is required. As the name suggests, `ElevateImmediateCommands` will _always_ require mfa, and should therefore be used sparingly, especially if the mfa method is onerous (e.g. `totp`).




