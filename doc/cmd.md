# Using EEL as Command Line Tool

## Parameters

* in - (single) incoming event string surrounded by single quotes
* inf - (single) incoming event as file
* tf - JSON transformation as string surrounded by single quotes (one of tf or tff is mandatory)
* tff - JSON transformation as file
* istbe - boolean flag "is transformation by example?" (default true)

The transformation parameter tf/tff accepts both raw transformations (like in most of the examples below)
and transformations wrapped in a handler configuration (the ones that are used by EEL in proxy mode).

## Examples

Process single event:

```
./eelsys -in='{"foo":"bar"}' -tf='{"Foo":"{{/foo}}"}' -istbe=true
```

Process single event from file:

```
./eelsys -inf=event.json -tf='{"{{/}}":"{{/}}","{{/uuid}}":"{{uuid()}}"}' -istbe=false
```

Process multiple events (one event per line):

```
cat multipleevents.json | ./eelsys -tf='{"{{/}}":"{{/}}"}' -istbe=false
```

You can even use entire transformation handler files:

```
./eelsys -in='{"foo":"bar"}' -tff=../../config-handlers/tenant1/default.json
```

Just for fun: Using the command line version of EEL to parse log output of proxy version of EEL:

```
./eelsys -config=../../config-eel/config.json -handlers=../../config-handlers/ | ./eelsys -tf='{"{{/}}":"{{/}}"}' -istbe=false
``` 