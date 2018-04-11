**Gopherbot** comes with brain methods allowing plugin authors to store information long-term information like a `TODO` list, or short-term contextual
information, such as a particular list item under discussion. An important supplement to this guide can be found in the example scripting
plugins in the `plugins/` directory, and the `links` and `lists` *Go* plugins in the `goplugins` directory (from the source,
not included in the distributed `.zip` files).

Table of Contents
=================

  * [Memory Scoping](#memory-scoping)
  * [Long-Term Memories](#long-term-memories)
    * [Code Examples](#long-term-memory-code-examples)
    * [Sample Transcript](#long-term-memory-sample-transcript)
  * [Short-Term Memories](#short-term-memories)
    * [Method Summary](#method-summary)
    * [Code Examples](#short-term-memory-code-examples)
    * [Sample Transcript](#short-term-memory-sample-transcript)

# Memory Scoping
Long-term memories are scoped per-plugin by key, and so the data is not shareable between plugins. Short-term memories are stored
for each user/channel combination, and can be freely shared between plugins.

# Long-Term Memories
The following methods are available for manipulating long-term memories:
* `CheckoutDatum(key, RWflag)` - returns a complex data item (memory) with a short-term exclusive lock on the datum if RW is `true`
* `CheckinDatum(memory)` - signals the robot to release the lock without updating
* `UpdateDatum(memory)` - updates the memory and releases the lock

## Long-Term Memory Code Examples
The memory stored can be an arbitrarily complex data item; a hash, array, or combination - anything that can be serialized to/from
JSON. The example plugins for **Python**, **Ruby** and **PowerShell** all implement a *remember* function that remembers a list (array)
of items. For the code examples, we'll start with a memory whose key is `memory`, and has two items defined by this snippet of JSON:
```json
["the Alamo", "Ferris Bueller"]
```

The examples will check if the memory exists, add "the answer is 42", and then update the memory.

Note that long-term memory commands aren't current implemented for `bash`.

### Python
```python
memory = bot.CheckoutDatum("memory", True)
if memory.exists:
    memory.datum.append("the answer is 42")
else:
    memory.datum = [ thing ]
ret = bot.UpdateDatum(memory)
if ret != Robot.Ok:
    bot.Say("Uh-oh, I must be gettin' old - having memory problems!")
```

### Ruby
Note: the Ruby example is a little more complicated; adapted from plugins/rubydemo.rb.

```ruby
memory = bot.CheckoutDatum("memory", true)
remembered = false
if memory.exists
  if memory.datum.include?(thing)
    bot.Say("That's already one of my fondest memories")
    bot.CheckinDatum(memory)
  else
    remembered = true
    memory.datum.push("the answer is 42")
  end
else
  remembered = true
  memory.datum = [ "the answer is 42" ]
end
if remembered
  ret = bot.UpdateDatum(memory)
  if ret != Robot::Ok
    bot.Say("Dang it, having problems with my memory")
  end
end
```

### PowerShell
```powershell
$memory = $bot.CheckoutDatum("memory", $TRUE)
if ($memory.exists) {
  $memory.Datum += "the answer is 42"
} else {
  [String[]] $memory.Datum = @( "the answer is 42" )
}
$ret = $bot.UpdateDatum($memory)
if ($ret -ne "Ok") {
  $bot.Say("I'm having a hard time remembering things")
}
```

## Long-Term Memory Sample Transcript

Using the `terminal` connector, you can see the `remember` function in action:
```
c:general/u:alice -> floyd, remember the answer is 42
general: Ok, I'll remember "the answer is 42"
c:general/u:alice -> |ubob
Changed current user to: bob
c:general/u:bob -> floyd, recall
general: Here's everything I can remember:
#1: the Alamo
#2: Ferris Bueller
#3: the answer is 42
```

From the transcript you can see that `alice` added the item to the list, which
was then visible to `bob`. The `links` and `lists` plugins are more useful, and
allow easy sharing of bookmark items or `TODO` lists, for example.

# Short-Term Memories

Short term memories are simple key -> string values stored for each user / channel combination, and expiring
after a time. The best example of this uses the built-in `links` and `lists` plugins, shown in this example
using the `terminal` plugin:
```
[gopherbot]$ ./gopherbot -l /tmp/bot.log -c cfg/term/
Terminal connector running; Use '|C<channel>' to change channel, or '|U<user>' to change user
c:general/u:alice -> link tuna casserole to https://www.allrecipes.com/recipe/17219/best-tuna-casserole/, floyd
general: Link added
c:general/u:alice -> add it to the dinner meals list
c:general/u:alice -> floyd
general: Ok, I added tuna casserole to the dinner meals list
c:general/u:alice -> floyd
general: Yes?
c:general/u:bob -> floyd, pick a random item from the dinner meals list
general: Here you go: tuna casserole
c:general/u:bob -> look it up, floyd
general: Here's what I have for "tuna casserole":
https://www.allrecipes.com/recipe/17219/best-tuna-casserole/: tuna casserole
```

Here, the robot is using short-term memories several times. When I forgot to address my command to the robot, the command
`add it to the dinner meals list` was stored in short-term memory for user `alice` in the `general` channel; then, when
I typed the robot's name, it checked short-term memory for the last thing `alice` said (stored automatically). Then,
the `links` plugin stored `tuna casserole` in the `item` short-term contextual memory; when I used `it` in the lists command,
the lists plugin checked the `item` short-term memory (see `contexts` in the plugin config for `lists`) and
substituted the value from the short-term memory.

## Method Summary
These methods are available for short-term memories:
* `Remember(key, value)` - associate the string `value` to `key`, always returns `Ok`
* `RememberContext(context, value)` - store a short-term contextual memory for use with other plugins
* `Recall(key)` - return the short-term memory associated with `key`, or the empty string when the memory doesn't exist

Note that the short-term memory API doesn't have complicated return values. They are always stored in the robot's working
memory and never persisted, and expire after several minutes - so plugins should always be prepared to get blank return
values.

## Short-Term Memory Code Examples

Note that the short-term memory API is super trivial, so I didn't go to great lengths to provide
detailed examples. The `bash` example comes from the `bashdemo.sh` plugin.

### Bash
```bash
Remember "$1" "$2"
Say "I'll remember \"$1\" is \"$2\" - but eventually I'll forget!"
```
```bash
MEMORY=$(Recall "$1")
if [ -z "$MEMORY" ]
then
	Reply "Gosh, I have no idea - I'm so forgetful!"
else
	Say "$1 is $MEMORY"
fi
```

### Python
```python
bot.Remember(key, value)
```
```python
mem = bot.Recall(key)
```

### Ruby
```ruby
bot.Remember(key, value)
```
```ruby
mem = bot.Recall(key)
```

### PowerShell
```powershell
$bot.Remember($key, $value)
```
```powershell
$mem = $bot.Recall($key)
```

## Short-Term Memory Sample Transcript
Here you can see the robot's short term memories of Ferris Bueller in action (using
the `bashdemo.sh` plugin):

```
[gopherbot]$ ./gopherbot -l /tmp/bot.log -c cfg/term/
Terminal connector running; Use '|C<channel>' to change channel, or '|U<user>' to change user
c:general/u:bob -> floyd, what is Ferris Bueller
general: @bob Gosh, I have no idea - I'm so forgetful!
c:general/u:bob -> floyd, store Ferris Bueller is a Righteous Dude
general: I'll remember "Ferris Bueller" is "a Righteous Dude" - but eventually I'll forget!
c:general/u:bob -> floyd, what is Ferris Bueller
general: Ferris Bueller is a Righteous Dude
c:general/u:bob -> |ualice
Changed current user to: alice
c:general/u:alice -> floyd, what is Ferris Bueller
general: @alice Gosh, I have no idea - I'm so forgetful!
c:general/u:alice -> |ubob
Changed current user to: bob
c:general/u:bob -> floyd, what is Ferris Bueller
general: Ferris Bueller is a Righteous Dude
```