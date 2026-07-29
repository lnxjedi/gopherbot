# Environment-Scoped Secrets and Variables

Deployment data belongs only to the custom robot:

- `custom/conf/variables/common.yaml`
- `custom/conf/variables/<GOPHER_ENVIRONMENT>.yaml`

Common loads first; the selected environment overrides it. There is no
installed variables layer because engine defaults must not own deployment
secrets.

`{{ secret "NAME" }}` decrypts a declared secret and
`{{ variable "NAME" }}` reads a plaintext value during template expansion.
Missing references fail config load. Legacy `{{ decrypt "..." }}` fails with a
migration hint so ciphertext cannot remain scattered through config.

This structure exists to keep development and production credentials separate
while preserving explicit reviewable references. It does not broaden extension
access: attaching a secret/`ParameterSet` to an extension is still an
administrator grant, and generic Robot methods must not enumerate secret-bearing
config.

Launcher `GOPHER_*` values override private env files but are scrubbed from the
real process environment. `.env` contains the outer encryption key and must be
owner-readable only, especially under UID-only privsep.

Config migrations must update custom examples, `robot.skel/`, and
`UPGRADING-v3.md` together.
