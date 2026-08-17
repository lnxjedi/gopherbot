# Fail Pipelines

Fail tasks run only when the primary stage ends in failure.

## Ordering

Unlike final tasks, fail tasks run in the order they were added.

## Good use cases

- send an explicit failure notification
- gather debugging information
- alert another system
- leave breadcrumbs for an operator before cleanup runs
