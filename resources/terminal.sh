#!/bin/bash

# terminal.sh - run $BOTNAME in local terminal mode
echo "Setting 'GOPHER_PROTOCOL' to 'terminal' and logging to ./robot.log..."
GOPHER_PROTOCOL=terminal ./gopherbot -l robot.log
