# Deploying Gopherbot to Heroku

> NOTE: This is somewhat experimental. This README is mostly to just record the steps taken to deploy a **Gopherbot** robot. If it turns out to be a good solution, it will eventually become a chapter section in the online manual.

## Requirements

This document assumes you already have a [Heroku](https://heroku.com) account and the [Heroku CLI](https://devcenter.heroku.com/articles/heroku-cli) installed. Note that at the time of this writing, a personal unverified Heroku account gives you 550 dyno hours per month - not enough to run a full-time robot. Verifying the account by adding a credit card gives an additional 450 hours.

You should have already set up your robot, and have an environment file (e.g. `.env`) with all the `GOPHER_*` vars.

The steps assume you have the gopherbot dist archive unzipped in `/opt/gopherbot`; if not, grab the contents of `github.com/lnxjedi/gopherbot/resources/heroku` and stick it somewhere convenient.

## Steps

In this brief tutorial I'll be setting up my `clu` robot with a Heroku app name of `linux-jedi-clu`.

1. Create the app:
```
$ heroku apps:create linux-jedi-clu
Creating ⬢ linux-jedi-clu... done
https://linux-jedi-clu.herokuapp.com/ | https://git.heroku.com/linux-jedi-clu.git
```

2. Set config vars using `resources/heroku/heroku-app-vars.sh`:
```
$ /opt/gopherbot/resources/heroku/heroku-app-vars.sh linux-jedi-clu
Using '.env' for env vars ...
Setting GOPHER_ENCRYPTION_KEY and restarting ⬢ linux-jedi-clu... done, v3
GOPHER_ENCRYPTION_KEY: <redacted>
Setting GOPHER_CUSTOM_REPOSITORY and restarting ⬢ linux-jedi-clu... done, v4
GOPHER_CUSTOM_REPOSITORY: git@github.com:parsley42/clu-gopherbot.git
Setting GOPHER_PROTOCOL and restarting ⬢ linux-jedi-clu... done, v5
GOPHER_PROTOCOL: slack
Setting GOPHER_DEPLOY_KEY and restarting ⬢ linux-jedi-clu... done, v6
GOPHER_DEPLOY_KEY: -----BEGIN_OPENSSH_PRIVATE_KEY-----:<redacted ...>
```

3. Copy the trivial Heroku Dockerfile to your current directory and edit if desired (to e.g. add extra tools/libraries):
```
$ cp /opt/gopherbot/resources/heroku/Dockerfile .
```

4. Build and push your worker container to heroku:
```
$ heroku container:push worker -a linux-jedi-gopherbot
=== Building worker (/home/parse/clu/Dockerfile)
Sending build context to Docker daemon  3.072kB
Step 1/3 : FROM quay.io/lnxjedi/gopherbot:latest
 ---> 59b05951b009
Step 2/3 : ENTRYPOINT [ "/opt/gopherbot/gopherbot" ]
 ---> Using cache
 ---> b8b13f6cb7c2
Step 3/3 : CMD [ "-plainlog" ]
 ---> Using cache
 ---> 85aa64b988cf
Successfully built 85aa64b988cf
Successfully tagged registry.heroku.com/linux-jedi-clu/worker:latest
=== Pushing worker (/home/parse/clu/Dockerfile)
The push refers to repository [registry.heroku.com/linux-jedi-clu/worker]
49b6043f8f8c: Layer already exists
173594a0ff26: Layer already exists
d20418061ae8: Layer already exists
ef9d19b874b3: Layer already exists
1c6efb4cbd71: Layer already exists
d4dfaa212623: Layer already exists
cba97cc5811c: Layer already exists
0c78fac124da: Layer already exists
latest: digest: sha256:7f4b81003365221653168c3c0217f29feabd997fe4b55e913e61fdc68b6fd69e size: 1995
Your image has been successfully pushed. You can now release it with the 'container:release' command.
```

5. When the command completes, release your worker container:
```
$ heroku container:release worker -a linux-jedi-clu
Releasing images worker to linux-jedi-clu... done
```

6. (Optional) To watch your robot starting up, you might want to open a new terminal and tail the logs for your app:
```
$ heroku logs -a linux-jedi-clu --tail
2021-03-31T19:34:20.122622+00:00 app[api]: Release v1 created by user parsley@linuxjedi.org
...
2021-03-31T20:04:22.777642+00:00 app[api]: Deployed worker (85aa64b988cf) by user parsley@linuxjedi.org
```

7. Scale your worker to 1:
```
$ heroku ps:scale worker=1 -a linux-jedi-clu
Scaling dynos... done, now running worker at 1:Free
```

With any luck, your robot will start up, clone it's repository, and connect to your team chat. Verified with my log tail:
```
2021-03-31T20:08:26.429427+00:00 app[api]: Scaled to worker@1:Free by user parsley@linuxjedi.org
2021-03-31T20:08:44.079472+00:00 heroku[worker.1]: Starting process with command `-plainlog`
2021-03-31T20:08:44.861112+00:00 heroku[worker.1]: State changed from starting to up
2021-03-31T20:08:46.777803+00:00 app[worker.1]: Initialized logging ...
...
2021-03-31T20:08:51.988977+00:00 app[worker.1]: Info: Initializing plugin: echo
2021-03-31T20:08:51.989011+00:00 app[worker.1]: Info: Robot is initialized and running
```
