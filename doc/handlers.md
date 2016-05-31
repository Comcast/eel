# Transformation Handlers

The following sections describe in detail the features of the JSON template language and how it can be used to
configure event transformations in EEL. Often there is more than one way to achieve the same result.

## Transformation Handler Configuration

Transformation handlers are used to tell EEL if and how to process JSON events it receives and where to
forward them to.

Each transformation handler is stored in a single JSON file. You can have one or more transformation handlers
and the handler files are organized in a folder structure by tenant like this:

```
config-handlers
 |
  --- tenant1
 |      |
 |       --- handler1.json
 |       --- handler2.json
 |       --- handler3.json
  --- tenant2
        |
         --- handler_a.json
         --- handler_b.json
         ...
 ...
```

The handler configuration folder can be configured with the EEL command line parameter `-handlers`, the default location
is `config-handlers`. Under that folder there is one folder per tenant. If you don't care about multi-tenancy just
create one tenant folder (for example `tenant1`) and place all your transformation configurations in there.

As a typical example consider this [../config-handlers/tenant1/default.json](../config-handlers/tenant1/default.json)
transformation handler which may be used as a template for your custom transformation handlers.

```
{
    "Version": "1.0",
    "Name": "Default",
    "Info": "Default handler for everything. Doesn't apply any transformation and forwards everything unchanged to Endpoint/Path.",
    "Active": true,
    "Match": null,
    "IsMatchByExample": false,
    "TerminateOnMatch": true,
    "Transformation": {
        "{{/}}": "{{/}}"
    },
    "IsTransformationByExample": false,
    "Path": "target",
    "Protocol": "http",
    "Verb": "POST",
    "Endpoint": "http://localhost:8082",
    "HttpHeaders": {
        "X-B3-TraceId": "{{traceid()}}",
        "X-Tenant-Id": "{{tenant()}}"
    }
}
```

The most important parameters are `Transformation` and `Match`.

* [Parameters for Meta Data](#parameters-for-meta-data)
* [Parameters for Event Transformation](#parameters-for-event-transformation)
* [Parameters for Endpoint Configuration](#parameters-for-endpoint-configuration)
* [Parameters for Handler Selection](#parameters-for-handler-selection)
* [Parameters for Event Filtering](#parameters-for-event-filtering)
* [Handler Resolution and Multi-Tenancy Support](#handler-resolution-and-multi-tenancy-support)

### Parameters for Meta Data

#### Version

Version of this transformation handler. This field is mandatory but arbitrary values are allowed (except for "").

#### Name

Unique name of the transformation handler. Mainly used for logging. This field is mandatory.

#### Info

Detailed description of transformation handler. This field is optional.

#### Active

If set to false the transformation handler will be disabled. Usually set to true.

### Parameters for Event Transformation

This section details how to use JPath expressions to describe structural JSON event transformations. There are
two different flavors of syntax you can choose from, "by path" and "by example". Depending on the use case
one is usually more elegant than the other. Choosing the syntax style is done by setting the
boolean flag `IsTransformationByExample`.

#### Transformation (By-Path-Syntax)

Describes a structural transformation to be applied to an incoming event before forwarding it.

_*Example 1:*_ Identity transformation (doesn't change a thing).

```
"IsTransformationByExample" : false,
"Transformation" : {
   "{{/}}":"{{/}}"
}
```

_*Example 2:*_ Take everything under `content` and place it under a new element labeled `event`. In addition a
new element `"sync":true` is injected.

```
"IsTransformationByExample" : false,
"Transformation" : {
   "{{/event}}":"{{/content}}",
   "{{/sync}}":true
}
```

Input event:

```
{
  "content" : {
    "foo" : "bar"
  }
}
```

Output event:

```
{
  "event" : {
    "foo" : "bar"
  },
  "sync" : true
}
```

_*Example 3:*_ Filter everything except for `/content/accountId` and `/content/adapterId`.

```
"IsTransformationByExample" : false,
"Transformation": {
   "{{/content/accountId}}":"{{/content/accountId}}",
   "{{/content/adapterId}}":"{{/content/adapterId}}"
}
```

Input event:

```
{
  "content" : {
    "accountId" : "123",
    "adapterId" : "xyz",
    "timestamp" : 12345
  }
}
```

Output event:

```
{
  "content" : {
    "accountId" : "123",
    "adapterId" : "xyz"
  }
}
```

_*Example 4*_: Injecting JSON response from external service into event.

```
"IsTransformationByExample" : false,
"Transformation" : {
  "{{/comcast/quote}}":"{{eval('/query/results/quote/LastTradePriceOnly','{{curl('GET', 'http://query.yahooapis.com/v1/public/yql?q=select%20*%20from%20yahoo.finance.quotes%20where%20symbol%20in%20%28%22CMCSA%22%29%0A%09%09&env=http%3A%2F%2Fdatatables.org%2Falltables.env&format=json')}}')}}"
}
```

Output event:

```
{
	"comcast": {
		"quote": "54.61"
	}
}
```

Some examples for invalid transformations:

_*Bad Example 1*:_

JPath expression does not start with `/`.

```
"IsTransformationByExample" : false,
"Transformation" : {
   "{{event}}":"{{/content}}",
   "{{/sync}}":true
}
```

_*Bad Example 2:*_

JPath expression is missing a closing bracket `}`.

```
"IsTransformationByExample" : false,
"Transformation" : {
   "{{/event}}":"{{/content}",
   "{{/sync}}":true
}
```

#### Transformation (By-Example-Syntax)

All examples above describe transformations as a collection of JPath-to-JPath
mappings (or "by path") and thus require `IsTransformationByExample` to be set to `false`.
Another way of describing a transformation is "by example" in which case this parameter must be set to `true`.

Here we describe a transformation directly as the desired output document. This is useful when the output
event is structurally very different from the input event, or, if the output event contains complex JSON arrays
which are hard to describe otherwise.

_*Example:*_

```
"IsTransformationByExample" : true,
"Transformation" : {
   "foo": [ "a", "b", "{{/foo/bar}}" ]
}
```

Input event:

```
{
  "foo" : {
    "bar" : "c"
  }
}
```

Output event:

```
{
  "foo" : [ "a", "b", "c"]
}
```

Note that Match, Transformation and Filter all support both by-path and by-example syntax.

#### CustomProperties

Optional. Can be used to define local variables. For example, to avoid duplicate (expensive) calls to external services.
Or, to define complex function parameters such as patterns for the transform() function.

Building on Example 4 in the Transformation section above we could also do this:

_*Example:*_

```
"IsTransformationByExample" : false,
"Transformation" : {
  "{{/comcast/quote}}" : "{{eval('/LastTradePriceOnly','{{prop('comcast')}}')}}",
  "{{/comcast/name}}" : "{{eval('/LastTradePriceOnly','{{prop('comcast')}}')}}"
},
"CustomProperties" : {
  "comcast":"{{eval('/query/results/quote','{{curl('GET', 'http://query.yahooapis.com/v1/public/yql?q=select%20*%20from%20yahoo.finance.quotes%20where%20symbol%20in%20%28%22CMCSA%22%29%0A%09%09&env=http%3A%2F%2Fdatatables.org%2Falltables.env&format=json')}}')}}"
}
```

Output event:

```
{
	"comcast": {
		"quote": "54.61"
	}
}
```

### Parameters for Endpoint Configuration

Most of the endpoint parameters are optional. If not set EEL will http POST transformed events to the
endpoint(s) configured in [../config-eel/config.json](../config-eel/config.json).

#### Protocol

Protocol to use for sending transformed events downstream. Default is `http` and for now this is the only
protocol supported. EEL comes with a pluggable publisher framework and the idea is to support protocols other
than http in the future.

#### Verb

Http verb to use for sending transformed events downstream. Default is `POST`.

#### Endpoint

Optional. Endpoint to send transformed events to. If not set the default endpoint(s) configured
in [../config-eel/config.json](../config-eel/config.json) will be used instead. May be a string (single endpoint)
or a JSON array (for multiple endpoints).

#### Path

Optional. JPath expression describing how to generate a path relative to the endpoint to forward events to. In the above
example `Path` is `target`. Thus, this handler will forward incoming events to
`http://localhost:8082/target`. Note that the path must be a string, therefore the JPath expression
must result in string. May be an array of multiple JPath expressions for fanning out to multiple paths.

Example 1: Use the `id` element of the JSON event if present, otherwise use a blank path `""`.

```
"Path" : "{{/content/id}}"
```

Example 2: Always use the constant string `"csv"` as path.

```
"Path" : "cvs"
```

Example 3: Concatenate the constant string `"csv-"` and the `id` field from the incoming event as path.

```
"Path" : "csv-{{/content/id}}"
```

Example 4: Generate more than one path. This is yet another way to configure EEL to fan out
and publish more than one event to an endpoint for a single event received.

```
"Path" : ["csv","{{/content/id}}"]
```

#### HttpHeaders

Arbitrary list of HTTP header key-value pairs to be set when forwarding events. Values can
be simple constants or complex JPath expressions. Headers are optional.

One header that could be configured here is the
trace ID as defined in [../config-eel/config.json](../config-eel/config.json) for logging.
For compliance with the Zipkin framework this header is set to "X-B3-TraceId" by default.

_*Example 1:*_ Using elements from the incoming event for generating a trace header.

```
"HttpHeaders": {
  "X-B3-TraceId": "{{/sequence}}-{{/timestamp}}",
  "X-Tenant-Id": "tenant()"
}
```

_*Example 2:*_ Using EEL built-in functions to generate a unique trace header.

```
"HttpHeaders": {
  "X-B3-TraceId": "{{uuid()}}-{{time()}}",
  "X-Tenant-Id": "tenant()"
}
```

_*Example 3:*_ Using Zipkin-compliant trace id if present.

```
"HttpHeaders": {
  "X-B3-TraceId": "{{traceid()}}",
  "X-Tenant-Id": "tenant()"
}
```

### Parameters for Handler Selection

#### Match and IsMatchByExample

`Match` is used to match incoming events to transformation handlers (provided you have more than one
transformation handler configured). If an event matches all key-value pairs described in this section,
the transformation handler will be chosen for processing the event.

Example 1: Default handler matches any event regardless of payload (this is what the handler above does).

```
"Match": null
```

Example 2: Only process event with this transformation handler if key `foo` is present with value `bar`.

```
"Match": {
    "{{/foo}}": "bar"
},
IsMatchByExample: false
```

Example 3: Only process event matching three distinct key-value pairs.

```
"Match": {
    "{{/content/type}}": "ALARM",
    "{{/content/adapter}}": "xyz",
    "{{/content/accountId}}": "123456789"
},
IsMatchByExample: false
```

Example 4: Similar to example 3 but using by-example-syntax. Makes also use of wild card `*` and boolean or `||`.

```
"Match": {
    "content": {
      "type": "ALARM",
      "adapter": "xyz1||xyz2",
      "accountId": "*"
    }
},
IsMatchByExample: true
```

#### TerminateOnMatch

Usually set to true.

If this configuration matches an incoming event and TerminateOnMatch is set to true, the handler
is used and processing of the event stops. Otherwise, EEL will keep looking for other matching transformation
handlers. EEL will process transformation handlers with a "stronger" match first. In the section above, EEL
would favor a handler matching Example 4 over a handler matching Example 2 over a handler matching Example 1.

See the section about handler resolution below for details.

### Parameters for Event Filtering

Once a transformation handler matches an incoming event, it can decide to discard the event by filtering.
All event filtering parameters are optional.

#### Filters

Filters are optional. You can have zero, one or more filters.
`Filters` describes a list of patterns that must be matched by the incoming event. If the event does not match all of the patterns,
EEL will discard the event and not forward it. Filters lets you choose between by-path syntax and
by-example syntax using the `IsFilterByExample` parameter. If the parameter `IsFilterInverted` is
set to `true`, events will be filtered if they do NOT match the pattern described by `Filter`.
If the parameter `FilterAfterTransformation` is set to `true`, the filter will be applied to the
outgoing event (after the transformation), otherwise it will be applied to the incoming event
(before the transformation).

_*Example 1:*_ Let everything come through.

```
"Filters" : [
  {
    "Filter" : {
      "*" : "*"
    },
    "IsFilterByExample" : true,
    "IsFilterInverted" : false,
	"FilterAfterTransformation" : false
  }
]
```

_*Example 2:*_ Only process events with `id` present regardless of its value.

```
"Filters" : [
  {
    "Filter" : {
       "content" : {
         "id" : "*"
       }
    },
    "IsFilterByExample" : true,
    "IsFilterInverted" : false,
	"FilterAfterTransformation" : false
  }
]
```

_*Example 3:*_ Only process events with `id` present and value `"123"`.

```
"Filters" : [
  {
    "Filter" : {
       "content" : {
          "id" : "123"
       }
    },
    "IsFilterByExample" : true,
    "IsFilterInverted" : false,
	"FilterAfterTransformation" : false
  }
]
```

_*Example 4:*_ Only process events with `id` present and value `"123"` or `"xyz"`.

```
"Filters" : [
  {
    "Filter" : {
       "content" : {
          "adapterId" : "123||xyz"
       }
    },
    "IsFilterByExample" : true,
    "IsFilterInverted" : false,
	"FilterAfterTransformation" : false
  }
]
```

_*Example 5:*_ Only process events where `message` is `found_problem`.

```
"Filters": [
	{
		"Filter": {
			"{{equals('{{/message}}','found_problem')}}": false
		},
	    "IsFilterByExample" : false,
	    "IsFilterInverted" : false,
		"FilterAfterTransformation" : false
	}
]
```

## Handler Resolution and Multi-Tenancy Support

EEL is designed to handle JSON events for multiple tenants, each of which may require different event transformations, endpoints etc.

For a given TenantId `tid`, EEL considers all transformation handlers it finds in the tenant folder `config-handlers/tid/`. Each transformation handler is represented by a JSON encoded file located in that folder.

The example handler above is stored in a file `config-handlers/tenant1/default.json`.

Note that different tenants can express an interest in the same type of event. Therefore, EEL can be configured to fan out and send more than one event to different endpoints for every event it receives.

When EEL receives a JSON event, transformation handler resolution is performed as follows.

For each tenant folder EEL executes the following algorithm:

* If there is a transformation handler that has all key-value pairs in its `Match` section in common with the incoming event, process the event using this handler. If more than one transformation handler match the event, then the handler with the strongest match (most matching key-value pairs) will be used first. If such handler has `TerminateOnMatch` set to `true` , processing ends here. Otherwise, processing continues with weaker matching handlers.
* If the event has not been handled by any handler and there is a default handler `"Match" : null`, then the default handler will be applied.
* If the event has not been handled yet it will be discarded for this tenant.

A note on filtering events with `Filter`: If the matching transformation handler has a filter clause that causes
the event to be filtered, this will only work if the handler also has `TerminateOnMatch` set to `true`. Otherwise another transformation handler with a weaker match or a default handler (if present) may handle and forward the event.
