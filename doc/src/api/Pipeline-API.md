**Gopherbot** takes a slightly different approach to creating pipelines; pipelines are created by Add/Fail/Final Job/Command/Task family of methods, rather than by fixed configuration directives. This allows flexible configuration of pipelines if desired for e.g. a CI/CD application, or dynamic generation of pipelines based on logic at runtime.

Until more documentation is written, see:
- [The Gopherbot Pipeline Source](https://github.com/lnxjedi/gopherbot/blob/master/.gopherci/pipeline.sh)
- [The Configuration repository for Floyd, the robot that builds Gopherbot](https://github.com/parsley42/floyd-gopherbot)

Table of Contents
=================

  * [AddTask](#addtask)
  * [SetParameter](#setparameter)

## AddTask
The `AddTask` method ... TODO: finish me!

### Bash
```bash
AddTask "echo" "hello, world"
```

### Python
```python
```

### Ruby

```ruby
```

### PowerShell
```powershell
$ret = $bot.AddTask("echo", @("hello", "world"))
```

## SetParameter
