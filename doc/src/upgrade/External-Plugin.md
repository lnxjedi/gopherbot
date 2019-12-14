# External Plugin Configuration

ExternalPlugins were formerly a list, but are now a hash, so:
```yaml
ExternalPlugins:
- Name: chuck
  Path: plugins/chuck.rb
```
   becomes:
```yaml
ExternalPlugins:
  "chuck":
    Path: plugins/chuck.rb
```

This allows all default configuration in the installation to be flexibly modified for a particular robot, using the new configuration merging.