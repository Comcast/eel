# Built-in Functions

EEL provides a number of built-in functions that can be used in JPath expressions.

Features and limitations:

* In a transformation handler configuration, JPath expressions (including functions) can be used almost anywhere, including in the Path, HttpHeader, Transformation, Endpoint, CustomProperties, Filter and Match sections.
* Functions can only have string parameters surrounded by single quotes or no parameters at all. Example: `{{ident('foo')}}`
* There can be multiple function calls per JPath expression. Function calls may be concatenated and/or nested. Example: `foo-{{uuid()}}-{{ident('{{uuid()}}')}}`
* Function return values can be of type string, float, integer, bool, map or array. If multiple function calls are combined, the results will be auto-converted to string type before concatenation. Map or array return values should only be used inside of transformations but not for endpoints, paths or HTTP headers (those should be of type string).

## Example: ID Mapping with External Services using the curl() Function

Suppose we want to call an external mapping service to map an id given in the JSON event as `{{/content/id}}`
to a different type of id and inject the mapped ID into the JSON event ad `{{/content/mappedId}}`.
Further assume the external service can be called like this `http://mappingservice/someid` and returns a
JSON response like this:

```
{
  "originalId" : "123",
  "mappedId" : "xyz"
}
```

In the following are some examples of how the built-in function `curl()` can be used to obtain and inject
the mapped ID in a transformation.

_Example 1:_

```
"{{/content/mappedId}}" : "{{curl('GET', 'http://mappingservice/{{/content/id}}')}}"
```

Here the function "curl" is called to HTTP GET from the mapping service passing in the ID from the event as part
of the URL. Note that JPath expression '{{/content/id}}' will be evaluated BEFORE executing the function. Finally
the payload returned by the external service will be used verbatim as `mappedId`.  

_Example 2:_

```
"Path" : "{{eval('/mappedId','{{curl('GET', '{{prop('ServiceUrl')}}/{{/content/id}}')}}')}}"
```

Same as Example 1 but since the external service returns a JSON document and we are only interested in one field
we use eval() to extract that field (mappedId) AFTER we have received a response from the external service.
Note also the use of the prop() function to look up the URL of the external service from custom properties.

_Example 3:_

```
"Path" : "{{curl('POST', 'http://mappingservice', '{ \"query\" : \"{{/content/id}}\"}')}}"
```

Similar to Example 1 but here we are assuming that `mappingservice` expects to receive a JSON encoded query via POST.
This requires escaping the double quotes in the payload. Note that the the single curly brackets {} for the JSON encoding
mix with the double curly brackets used for JPath expressions without problems (other than readability).


## Function Reference

### curl

Used to hit an external web service.

Syntax:

```
{{curl('<method>','<url>',['<payload>'],[<'headers'>],[<'retries'>])}}
```

Example:

```
{{curl('POST', 'http://foo.com/bar/json', '{{/content/userId}}')}}
```

Parameters:

* method - POST, GET etc.
* url - url of external service
* payload - payload to be sent to external service
* headers - optional header map
* retries - if true, applies retry policy as specified in config.json in case of failure

### uuid

Returns UUID string.

Syntax:

```
{{uuid()}}
```

### ident

Returns input parameter unchanged.

Syntax:

```
{{ident('<param>')}}
```

Example:

```
ident('foo')
```

### eval

Evaluates a JPath expression against the current document or document provided as parameter and returns the result.

Syntax:

```
eval('<path>', ['<doc>'])
```

Parameters:

* path - simple JPath expression
* doc - JSON document (optional, if not present the current document will be used instead)

Example:

```
{{eval('/content/comcastId')}}

```

Note that this is equivalent to the simplified JPath expression notation:

```
{{/content/comcastId}}

```

There is also an extended version of the eval() function allowing to pass in a JSON
document as additional parameter. The JPath expression will then be applied to that
document rather than the current document.

Example:

```
{{eval('/accountId', '{"accountId":"42"}')}}

```

### prop

Returns a custom property from the `CustomProperties` map in EEL's `config.json` or a `CustomProperties`
map in the handler configuration. Properties can be constants or JPath expressions.

Syntax:

```
{{prop('<key>')}}
```

Example:

```
{{prop('ServiceUrl')}}
```

### js

Executes arbitrary JavaScript code and returns a variable.

Syntax:

```
{{js('<jssource>', ['<js_output_variable>'], ['<js_input_key>'], ['<js_input_value>'], ...)}}
```

Parameters:

* jssource - Java Script source code
* js_output_variable - Java Script variable to return (can be omitted if source code is an expression)
* js_input_key / js_input_value - arbitrary number of key-value-pairs to pass in to Java Script (can also inject values directly into the Java Script source using JPath expressions)

Example 1:

```
{{js('result = 40+2; result += 2;', 'result')}}
```

Example 2:

```
{{js('40+{{/content/number}}')}}
```

### alt

Return the first non-blank parameter of a list of parameters.

Syntax:

```
{{alt('<p1>', '<p2>', ['<p3>'], ...)}}
```

Parameters:

* p1, p2, p3, ... - two or more parameters (example: if p1 is blank and p2 and p3 are not, then p2 will be returned)

Example:

```
{{alt('{{eval('/item','{{curl('GET', '{{prop('MoleculeMappingServiceUrl')}}{{/content/accountId}}')}}')}}')}}','{{/content/accountId}}')}}
```

Here the account ID is returned unmodified if we cannot map it to a Comcast GUID.

### len

Returns length of given object (string, array, map).

Syntax:

```
{{len('<object>')}}
```

Example:

```
{{len('[1,2,3]')}}
```

### regex

Apply regular expression to string and return (first) match if any.

Syntax:

```
{{regex('<string>', '<regex>', ['<all>'])}}
```

Parameters:

* string - string to for regex to operate on
* regex - regular expression
* all - optional, if true, concatenate all matches to one match, otherwise only return first match (if any)

Example:

```
{{regex('{{/content/_links/iot:account/href}}','[A-Z0-9]{10,}+')}}
```

Example 2:

```
{{regex('(650) 233-7344', '[0-9]+', 'true')}}
```

Result:

```
6502337344
```

### match

Apply regular expression to string and return true if there is at least one match, false otherwise.

Syntax:

```
{{match('<string>', '<regex>')}}
```

Example:

```
{{match('{{/content/_links/iot:account/href}}','[A-Z0-9]{10,}+')}}
```

### join

Join two JSON documents. Key conflicts will be resolved randomly.

Syntax:

```
{{join('<docA>','<docB>')}}
```

Example:

```
{{join('{{eval('/code/data')}}','{\"protocol\":\"apns\"}')}}
```

### format

Format human readable time strings from epoch ms.

Syntax:

```
format('<ms>',['<layout>'],['<timezone>'])
```

Layout must follow the golang spec and provide the format string by example using Mon Jan 2 15:04:05 MST 2006.

Parameters:

* ms - timestamp in milliseconds
* layout - time format by example following go conventions (https://golang.org/src/time/format.go)
* timezone - valid timezone in tz format, for example US/Pacific, US/Mountain, US/Central, US/Eastern

Example:

```
{{format('1439937356000','3:04pm', 'EST')}}
```

### and

Boolean and for one or more parameters.

Syntax:

```
{{and('<bool>', '<bool>', ...)}}
```

Example:

```
{{and('false', '{{ident('true')}}')}}
```

### or

Boolean or for one or more parameters.

Syntax:

```
{{or('<bool>', '<bool>', ...)}}
```

Example:

```
{{or('false', '{{ident('true')}}')}}
```

### not

Boolean not for one parameter.

Syntax:

```
{{not('<bool>')}}
```

Example:

```
{{not('{{equals('{{/foo}}', 'bar')}}')}}
```

### contains

Checks if current document contains another document.

Syntax:

```
{{contains('<doc1>', ['<doc2>'])}}
```

Parameters:

* doc1 - JSON document to be contained in event document
* doc2 - JSON document (optional, if present containment will be checked agains this document instead of event document)

Example:

```
{{contains('{{/content}}', '{{curl('GET', 'http://mappingservice/{{/content/id}}', '', '')}}')}}
```

### ifte

If condition then this else that.

Syntax:

```
{{ifte('<condition>','<then>',['<else>'])}}
```

Parameters:

* condition - boolean condition
* then - string used if condition true
* else - string used if condition false (optional)

Example:

```
{{ifte('{{equals('{{/data/name}}','')}}','','by {{/data/name}}')}}
```

### equals

Checks if current document is equal to JSON document provided as parameter. Equality is based on
JSON structural comparison. The two-parameter version of this function compares either two json
documents or two strings for equality.

Syntax:

```
{{equals('<doc1>',['<doc2>'])}}
```

Parameters:

* doc1 - JSON document 1
* doc2 - JSON document 2 (optional, if not present the current document will be used instead)

Example:

```
{{equals('{"foo":"bar"}','{"foo":"bar"}')}}
```

Alternate two-parameter version for string comparison:

Syntax:

```
{{equals('<string1>','<string2>')}}
```

Parameters:

* string1 - arbitrary string
* string2 - arbitrary string

Example:

```
{{equals('foo','{{/data/bar}}')}}
```

### transform

Applies named transformation.

Syntax:

```
{{transform('<name_of_transformation>', ['<doc>'],['<pattern>'],['<join>'])}}
```

Parameters:

* name_of_transformation - the transformation is selected by name from an optional Transformations map in the handler config
* doc - if no document is provided the transformation will be applied to the event document
* pattern - if present, transformation will only be applied to document if it matches the pattern
* join - if present, document will be joined with join prior to applying the transformation

Example:

```
{{transform('myt2', '{{/}}')}}
```

Example Transformations section in topic handler config:

```
"Transformations" : {
  "myt1" : {
    "Transformation" : {
      "{{/}}":"{{/content}}"
    },
    "IsTransformationByExample" : false
  },
  "myt2" : {
    "Transformation" : {
      "{{/id}}":"{{/content/accountId}}"
    },
    "IsTransformationByExample" : false
  }
},
```

### etransform

Selects appropriate handler for event, performs transformation and returns result. Only works if only one handler
matches the event and the transformation yields only a single result. Otherwise an error will be returned. etransform
is equivalent to but slightly more efficient than `curl http://localhost:8080/v1/sync/events`.

Syntax:

```
{{etransform('<doc>')}}
```

Parameters:

* doc - document to be transformed

Example:

```
{{etransform('{{/}}')}}
```

### ptransform

Selects appropriate handlers for event, performs transformations and publishes results. ptransform
is equivalent to but slightly more efficient than `curl http://localhost:8080/v1/events`.

Syntax:

```
{{ptransform('<doc>')}}
```

Parameters:

* doc - document to be transformed

Example:

```
{{ptransform('{{/}}')}}
```

### itransform

Same as transform() but applies named transformation iteratively if document is an array: In this case the transformation,
pattern and join parameters will all be applied to each element in the array.

Syntax:

```
{{transform('<name_of_transformation>', ['<doc>'],['<pattern>'],['<join>'])}}
```

Parameters:

* name_of_transformation - the transformation is selected by name from an optional Transformations map in the handler config
* doc - if no document is provided the transformation will be applied to the event document
* pattern - if present, transformation will only be applied to document if it matches the pattern
* join - if present, document will be joined with join prior to applying the transformation


### choose

Chooses elements from an array or map that match a given pattern.

Syntax:

```
{{choose('<doc>','<pattern>')}}
```

Parameters:

* doc - document containing array or map
* pattern - only elements matching the pattern (given in by-example syntax) are returned

Example:

```
{{choose('{{/}}','{{prop('mypattern')}}')}}

"CustomProperties" : {
	"mypattern" : {
		"type" : "door"
	}
}
```

### crush

Converts an array of arrays into a flat array.

Syntax:

```
{{crush('<doc>')}}
```

Parameters:

* doc - document to crush

Example:

```
{{crush('{{prop('mydoc')}}')}}

"CustomProperties" : {
	"mydoc" : [
		[1,2],[3,4]
	]
}
```

Result:

```
[1,2,3,4]

```

### true

Returns always true. Shorthand for equals('1','1').

Syntax:

```
{{true()}}
```

### false

Returns always false. Shorthand for equals('1','2').

Syntax:

```
{{false()}}
```

### time

Returns current time in milliseconds.

Syntax:

```
{{time()}}
```

Example return value:

```
1449194313
```

### tenant

Returns current tenant id.

Syntax:

```
{{tenant()}}
```

Example return value:

```
tenant1
```

### upper

Returns uppercase version of input string.

Syntax:

```
{{upper('<string>')}}
```

Example:

```
{{upper('fOo')}}
```

Example return value:

```
FOO
```

### lower

Returns lowercase version of input string.

Syntax:

```
{{lower('<string>')}}
```

Example:

```
{{lower('fOo')}}
```

Example return value:

```
foo
```

### substr

Returns substring of input string.

Syntax:

```
{{substring('<string>','<startidx>','<endidx>')}}
```

Example:

```
{{substr('fOo', 0, 1)}}
```

Example return value:

```
f
```

### traceid

Returns current trace id used for logging. If a Zipkin-compliant trace ID is passed in via HTTP header `X-B3-TraceId` it will be used,
otherwise a UUID will be generated automatically by EEL.

Syntax:

```
{{traceid()}}
```

### case

Simplification of a nested ifte(equals(),'foo', ifte(equals(...),...)) cascade.

Syntax:

```
{{case('<value_1>','<comparison_value_1>','<return_value_1>', '<value_2>','<comparison_value_2>','<return_value_2>,...,'<default>')}}
```

If value1 == comparison_value_1 then return return_value_1. Else, if value2 == comparison_value_2 then return return_value_2. Otherwise, return the default value.


Example:

```
{{case('{{/content/message}}', 'High WiFi','{{/content/device}} has returned to good Wi-Fi coverage','{{/content/message}}', 'Low WiFi','{{/content/device}} has returned to bad Wi-Fi coverage','{{/content/message}}')}}
```

### header

Returns http header value from incoming event by key. When the key parameter is omitted the entire header map will be returned.

Syntax:

```
{{header(['<key>'])}}
```

Example:

```
{{header('X-B3-TraceId')}}
```

### string

Convert an array of strings into a string using an optional separator between elements.

Syntax:

```
{{string('<doc>','<separator>')}}
```

Input:

```
["d1", "d2"]
```

Example:

```
{{string('{{/}}', '-')}}
```

Output:

```
d1 - d2
```
