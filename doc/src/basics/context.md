# Context

Commands can be configured to store certain matched fields as labeled context items, e.g. "item" or "list". This feature is somewhat experimental, but could occasionally be useful. A somewhat contrived example uses the "list" and "item" contexts with the aforementioned links and lists plugins:
```
c:general/u:alice -> !link broiled salmon to https://cooking.com/salmon.html
general: Link added
c:general/u:alice -> !add it to the dinner meals list
general: @alice I don't have a 'dinner meals' list, do you want to create it?
c:general/u:alice -> yes
general: Ok, I created a new dinner meals list and added broiled salmon to it
c:general/u:alice -> !link tuna casserole to https://cooking.com/tuna.html
general: Link added
c:general/u:alice -> !add it to the list
general: Ok, I added tuna casserole to the dinner meals list
```