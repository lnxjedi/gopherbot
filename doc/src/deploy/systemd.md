# Running with Systemd

One way of running your robot is to use a **systemd** unit file on a systemd-managed Linux host:

* Copy `resources/robot.service` to `/etc/systemd/system/<botname>.service` and edit with values for your system; you'll need to create a local user, and a directory for your robot that the user can write to
* Reload `systemd` with `systemctl daemon-reload`
* Enable the service with `systemctl enable <botname>`
* Place your robot's `.env` in the robot's home directory, mode `0400`, owned by the robot user; you can leave `GOPHER_PROTOCOL` commented out, since the value should be set in the `<botname>.service` file
* Start the service: `systemctl start <botname>`

That's it! Your robot should start and connect to your team chat.
