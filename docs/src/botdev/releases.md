# Releases

Given the relative infrequency of releases and newness of the development structure, this is still a manual process. Documented here for repeatability until it's automated.

## Cutting a new Minor Release - 2.x

To release e.g. v2.1:
1. Check out a new branch of the form `release-2.x`, e.g. `release-2.1`
   > Note that this just prevents an automatic build on push
1. Edit `main.go` and update `Version`
   * Automation note: the pipeline could be updated to pass this in to the build, similar to the commit.
1. Tag the release with the form `v2.x`, e.g. `v2.1`
1. Tell the configured build robot to `build gopherbot v2.x`
1. Check out `main` branch
1. Update the `Version` to `v2.(x+1)-pre`, e.g. `v2.2-pre`
