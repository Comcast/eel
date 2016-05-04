# Using EEL as Command Line Tool

## Parameters

* in - (single) incoming event string surrounded by single quotes
* inf - (single) incoming event as file
* tf - JSON transformation as string surrounded by single quotes (one of tf or tff is mandatory)
* tff - JSON transformation as file
* istbe - boolean flag "is transformation by example?" (default true)

Process single event:

```
./eelsys -in='{"foo":"bar"}' -tf='{"Foo":"{{/foo}}"}' -istbe=true
```

Process single event from file:

```
./eelsys -inf=event.json -tf='{"{{/}}":"{{/}}","{{/uuid}}":"{{uuid()}}"}' -istbe=false
```

Process multiple events (oen event per line):

```
cat multipleevents.json | ./eelsys -tf='{"{{/}}":"{{/}}"}' -istbe=false
```