# Using EEL as Command Line Tool

## Command Line Options

* in - (single) incoming event string surrounded by single quotes or as file prefixed with @
* tf - JSON transformation as string surrounded by single quotes (one of tf or tff is mandatory) or as file prefixed with @
* istbe - boolean flag "is transformation by example?" (default true)

The transformation parameter tf/tff accepts both raw transformations (like in most of the examples below)
and transformations wrapped in a handler configuration (the ones that are used by EEL in proxy mode).

## Examples

Process single event:

```
./eel -in='{"foo":"bar"}' -tf='{"Foo":"{{/foo}}"}' -istbe=true
```

Process single event from file:

```
./eel -in=@event.json -tf='{"{{/}}":"{{/}}","{{/uuid}}":"{{uuid()}}"}' -istbe=false
```

Process multiple events (one event per line):

```
cat multipleevents.json | ./eel -tf='{"{{/}}":"{{/}}"}' -istbe=false
```

You can even use entire transformation handler files:

```
./eel -in='{"foo":"bar"}' -tf=@config-handlers/tenant1/default.json
```

Just for fun: Using the command line version of EEL to parse log output of proxy version of EEL:

```
./eel | ./eel -tf='{"{{/}}":"{{/}}"}' -istbe=false
``` 