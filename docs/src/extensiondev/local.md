# Local Authoring Workflow

The normal v3 authoring loop looks like this:

1. run the engine from the robot home
2. connect over the local SSH connector
3. edit files under `custom/`
4. issue `reload`
5. test the command again

## Example loop

```bash
cd ~/robots/acme
/opt/gopherbot/gopherbot
```

In another terminal:

```bash
ssh -i /opt/gopherbot/resources/ssh-default/alice.key -p 4221 alice@localhost
```

Then:

- edit `custom/conf/plugins/hello.yaml`
- edit `custom/plugins/hello.lua`
- run `;reload`
- run `;hello`

That is the workflow the rest of this manual assumes.
