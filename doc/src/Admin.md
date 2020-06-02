# Robot Administration

**Gopherbot** robots are designed to be remotely administered, for common cases where a robot runs behind network firewalls, in virtual cloud networks, or in a container environment. Many of the frequently desired updates - such as changing the schedule of an automated job - can be safely and easily updated by pushing a commit to your robot's repository and instructing it to update. More significant updates can be tested locally by modelling with the **terminal** connector before committing and pushing.

This chapter outlines the common workflow for managing your robots remotely, and documents other important administrative tasks.