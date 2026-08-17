# Docker Example

A minimal container run looks like this:

```bash
docker container run --name acme-bot \
  --restart unless-stopped \
  --env-file .env \
  -e HOSTNAME="$(hostname)" \
  ghcr.io/lnxjedi/gopherbot:latest
```

## Notes

- Build and validate the robot locally before you do this.
- Keep `.env` out of image layers and source control.
- If your robot needs extra operating-system tools, build a derived image that adds them.
- `podman` works fine for the same pattern.
