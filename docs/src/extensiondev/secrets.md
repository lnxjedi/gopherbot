# Secrets and Parameters

Gopherbot gives you two main ways to supply sensitive values:

- encrypted values embedded in config with `{{ decrypt "..." }}`
- parameters passed to plugins, jobs, tasks, or namespaces

## Encrypt a value

```bash
gopherbot encrypt foobarbaz
```

Then use the returned blob in config:

```yaml
Parameters:
- Name: API_TOKEN
  Value: {{ decrypt "<encrypted-value>" }}
```

## Secure parameter handling

When `SecureParameters: true` is enabled, tasks may need to fetch secrets with `GetParameter(...)` instead of reading them directly from the process environment.

That is the safer v3 posture and the one new robots should expect.
