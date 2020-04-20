# Protocols

At different points during the development of **Gopherbot**, consideration was given to the possibility of being *multi-protocol*, allowing messages to come in from different protocols to a single running instance. This has not come to fruition, and is left as being 'under consideration'.

The primary use for the **Protocol** struct fields (and `GOPHER_PROTOCOL` environment variable) is being informative for extensions; this functionality should remain, and stay much the same. Otherwise, the multi-protocol question is left open.