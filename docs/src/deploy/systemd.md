# Run with systemd

`systemd` is the easiest long-running deployment target for most Linux robots.

## Recommended shape

1. Install the engine in a stable location such as `/opt/gopherbot`.
2. Create a dedicated robot home such as `/srv/robots/acme`.
3. Create a dedicated Unix user for that robot.
4. Put the robot's `.env` in the robot home with restrictive permissions.
5. Copy and adapt `resources/robot.service` or `resources/user-robot.service`.

## Minimal service pattern

```ini
[Service]
Type=simple
WorkingDirectory=/srv/robots/acme
ExecStart=/opt/gopherbot/gopherbot -plainlog
Restart=on-failure
TimeoutStopSec=600
```

## Notes

- Keep the engine install tree and the robot home separate.
- The robot home should be writable by the robot user.
- `TimeoutStopSec=600` is intentional; Gopherbot tries to finish in-flight pipelines during graceful shutdown.
- Prefer configuring protocol and behavior in `custom/conf/` and `.env` instead of stuffing everything into `Environment=` lines.

## Bring it up

```bash
sudo systemctl daemon-reload
sudo systemctl enable acme-bot
sudo systemctl start acme-bot
sudo systemctl status acme-bot
```
