# Installation Overview

Version 2 of **Gopherbot** introduced a new, unified and simplified install method for both host- and container- based installs. Whether creating an instance of a new robot, or running an existing robot in a new location, basic installation to a host consists of a few simple steps:

1. Un-zip or un-tar the download archive in `/opt/gopherbot`
2. Create a new empty directory for your robot; by convention `\<name\>-gopherbot`, e.g. `clu-gopherbot`
   1. Optionally, create a symlink to the binary, e.g.: `cd clu-gopherbot; ln -s /opt/gopherbot/gopherbot .`
3. Run the gopherbot binary from the new directory: `/opt/gopherbot/gopherbot` or `./gopherbot`