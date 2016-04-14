# EEL - A simple Proxy Service for JSON Event Transformation and Forwarding

## Motivation

It's simple - a single JSON event comes in and one (or maybe a few) transformed events get out.
Events are arbitrary JSON encoded payloads and they are typically sent around as
HTTP POSTs. EEL is stateless and therefore scales easily.

![](doc/eel.png)

In this example we removed the unwanted "detail" field from the event and adjusted
the schema slightly.

Converting JSON payload A from service S1 into JSON payload B for service S2 is a very common
operation we perform all the time in service oriented architectures. Here are some good reasons
for event transformation:

1. REDUCTION: Remove everything that's not needed and only pass the data actually required by the downstream service. This can provide performance benefits for the downstream service if processing is expensive and events are bloated.
1. CANONICALIZATION / CLEANUP: Adjust the structure of the JSON event to validate against a given schema. Typically this can be achieved by rearranging elements in the JSON event, renaming keys and possibly filtering out unwanted subsets of data.
1. META-TAGGING: Inject additional metadata into the JSON event. Examples are injecting UUIDs or timestamps (for example for log tracing), tagging or labeling events with `type` fields, calling external auxiliary services and injecting their output into the downstream payload (provided it is JSON).
1. MAPPING / SUBSTITUTION: Map one ID type to another via look up using an external lookup service.
1. FILTERING: Filter out unwanted events to reduce load for downstream services.
1. FANOUT: Forward a single incoming JSON event to multiple downstream services, either by applying the same transformation for all services or by performing different transformations.

You could write each of these transformations in just a few lines of go code, and we often do.
The downside is that transformations become engraved in code and cannot be changed easily.

EEL is offering a simple JSON template language combined with a proxy micro service to relay
and transform events between upstream and downstream services. The goals of the EEL template language are to be
simple and yet powerful. EEL can be the glue in a JSON based service oriented eco-system.

The syntax of the EEL transformation language is inspired by a few of the core concepts from XPath and XSLT applied to JSON.

## A Simple Example

Transformation handlers are used to tell EEL if and how to process JSON events it receives and where to
forward them to. Each transformation handler is stored in a single JSON file and there may be several such
handler files each of them taking care of a certain class of events. See [here](doc/handlers.md) for more details.

Edit the default transformation handler in [config-handlers/tenant1/default.json](config-handlers/tenant1/default.json)
so it looks like this (there's only a small change needed in the `Transformation` section).

```
{
    "Version": "1.0",
    "Name": "Default",
    "Info": "",
    "Active": true,
    "Match": null,
    "IsMatchByExample": false,
    "TerminateOnMatch": true,
    "Transformation": {
        "{{/event}}": "{{/}}"
    },
    "IsTransformationByExample": false,
    "Path": "",
    "Verb": "POST",
    "Endpoint": "http://localhost:8082",
    "HttpHeaders": {
      "X-B3-TraceId": "{{traceid()}}",
      "X-Tenant-Id": "{{tenant()}}"
    }
}
```

The center piece of the handler configuration is the `Transformation` section which uses [JPath](doc/jpath.md) expressions to describe the structural transformations to be performed on the incoming event before forwarding it to the downstream service. In this example we are telling EEL to take the entire payload unmodified and wrap it inside a new element called `event`.

Compile EEL.

EEL is implemented in go and you will need a [golang](https://golang.org/) environment to compile the code. However, you don't need any go coding skills to understand and author EEL templates!

The make file and shell scripts have been tested on Mac and Linux environments.

```
make all
```

Launch EEL as a service and start listening to incoming events.

```
./bin/starteel.sh
```

Send a test event to EEL.

```
curl -X POST --data '{ "message" : "hello world!!!" }' http://localhost:8080/proxy
```

Output:

```
{"status":"processed"}
```

You can check EEL's log output `eel.log` to see if and how the event is processed. You can also ask for a detailed
debug response using the `X-Debug` header.

```
curl -X POST -H 'X-Debug: true' --data '{ "message" : "hello world!!!" }' http://localhost:8080/proxy
```

Output:

```
[
	{
		"api": "http",
		"endpoint": "http://localhost:8082",
		"event.in": {
			"message": "hello world!!!"
		},
		"event.out": {
			"event": {
				"message": "hello world!!!"
			}
		},
		"handler": "Default",
		"headers": {
			"X-B3-TraceId": "1449343562055990060",
			"X-Tenant-Id": "tenant1"
		},
		"path": "target",
		"protocol": "http",
		"tenant.id": "tenant1",
		"topic": "",
		"tx.id": "053e4424-9b86-11e5-8eb1-a45e60edf673",
		"tx.traceId": "1449343562055990060",
		"url": "http://localhost:8082/target",
		"verb": "POST"
	}
]
```

Or, you can use the `/proc` instead of the `/proxy` endpoint to get an immediate response containing the transformed event as it would be forwarded to the downstream service.

```
curl -X POST --data '{ "message" : "hello world!!!" }' http://localhost:8080/proc
```

Output:

```
{
	"event": {
		"message": "hello world!!!"
	}
}
```

To review the current set of active transformation handlers call the health check API.

```
curl http://localhost:8080/health
```

Stop EEL.

```
./bin/stopeel.sh
```

## Exploring EEL Features

The unit tests are a good starting point to learn more about EEL features and look at some examples in detail.

Each test is contained in its own configuration folder, for example `eel/test/data/test01`. Each test folder
contains a complete set of handler configurations for EEL in the `handlers` subfolder (usually just one handler),
an example input event `in.json` and one or more expected output events `out.json`.

The tests can be executed using `go test`:

```
cd eel/test
go test -v
```

Or, you can launch EEL with the handler configurations for a specific test and send the `in.json` event manually.

```
cd eel/eelsys
./eelsys -config=../../config-eel/config.json -handlers=../test/data/test01/handlers > eel.log &
curl -X POST --data @../test/data/test01/in.json http://localhost:8080/proc
```

No | Name | Test Name | Description
--- |--- | --- | ---
0 | Identity Transformation | [TestDontTouchEvent](eel/test/data/test00) | Doesn't apply any transformation and forwards everything unchanged.
1 | Canonicalize | [TestCanonicalizeEvent](eel/test/data/test01) | Simple structural changes and array path selectors.
2 | External Lookup | [TestInjectExternalServiceResponse](eel/test/data/test02) | Get JSON data from external service and inject into payload.
3 | Transformation By Example | [TestTransformationByExample](eel/test/data/test03) | Describe transformation using by-example syntax rather than using by-path syntax.
4 | Named Transformations | [TestNamedTransformations](eel/test/data/test04) | Choose from several named transformations based on input data.
5 | Conditional Message Generation | [TestMessageGeneration](eel/test/data/test05) | Assemble message string with ifte() and equals().
6 | Handler Matching 1 | [TestTerminateOnMatchTrue](eel/test/data/test06) | Pick best matching handler and forward single event.
7 | Handler Matching 2 | [TestTerminateOnMatchFalse](eel/test/data/test07) | Pick all matching handlers and forward multiple events.
8 | Multi Tenancy | [TestMultiTenancy](eel/test/data/test08) | Handlers for different tenants or apps.
9 | Cascade | [TestSequentialHandlerCascade](eel/test/data/test09) | Cascade of multiple handlers which will be executed sequentially by using EEL recursively.
10 | Java Script For Everything Else | [TestJavaScript](eel/test/data/test10) | If you really can't avoid it, resort to Java Script.
11 | Handler Matching 3 | [TestMatchByExample](eel/test/data/test11) | Matching handlers using by-example syntax.
12 | Convert Headers To Payload | [TestHeaders](eel/test/data/test12) | Inject HTTP headers from upstream service into JSON event for downstream service.
13 | Custom Properties | [TestCustomProperties](eel/test/data/test13) | Custom properties in handlers for sharing data.
14 | Fan Out | [TestFanOut](eel/test/data/test14) | Send incoming event to several downstream services.
15 | Basic String Operations | [TestStringOps](eel/test/data/test15) | Uppercase, lowercase, substring.
16 | Named Transformations 2 | [TestNamedTransformations2](eel/test/data/test16) | Perform named transformation on external document.
17 | Contains | [TestContains](eel/test/data/test17) | Check if one JSON event is contained in another.
18 | Tenant Id Header | [TestMultiTenency2](eel/test/data/test18) | Pass in tenant id as HTTP header.
19 | Conditional Message Generation 2 | [TestMessageGeneration2](eel/test/data/test19) | Use case() function to simplify conditional string generation.
20 | Regex | [TestRegex](eel/test/data/test20) | Use regex() to evaluate regular expressions.
22 | Filter By Path | [TestFilterByPath](eel/test/data/test22) | Filter event after transformation using by-path syntax.
23 | Filter By Example | [TestFilterByExample](eel/test/data/test23) | Filter event after transformation using by-example syntax.
25 | Array Path Selector | [TestArrayPathSelector](eel/test/data/test25) | Select elements from arrays by index or by path.
27 | Iterate Over Array | [NamedTransformationsAndArrays](eel/test/data/test27) | Iterate over array and apply named transformation.
32 | Simple Types | [NamedTransformationsAndSimpleTypes](eel/test/data/test32) | Apply named transformation to a simple type.
34 | Join | [Join](eel/test/data/test34) | Merge two JSON documents.
41 | Partial Matching | [MatchPartialArrays3](eel/test/data/test41) | Match event against partial pattern.
46 | Filter Cascade | [FilterCascade](eel/test/data/test46) | Apply multiple filters.
47 | Choose From Array | [ChooseFromArray](eel/test/data/test47) | Choose elements from array by pattern.
48 | Named Transformation With Array And Pattern | [NamedTransformationWithArrayAndPattern](eel/test/data/test48) | Apply named transformation by pattern.
49 | Named Transformation With Array And Join | [NamedTransformationWithArrayAndJoin](eel/test/data/test49) | Apply named transformation with join.
51 | Complex Example | [ComplexExample](eel/test/data/test51) | A real world example.
53 | Crush | [Crush](eel/test/data/test53) | Flatten a deeply nested array.

## Further Reading

* [JSON Path Expressions](doc/jpath.md)
* [Transformation Handler Configuration](doc/handlers.md)
* [Function Reference](doc/functions.md)
* [Global Configuration Parameters](doc/configuration.md)
* [Debug Tools](doc/debug.md)
* [EEL Web API Reference](doc/apis.md)
* [EEL Transformations as Go Library](doc/lib.md)
