# JSON Path Expressions

JSON Path expressions (or JPath expressions) are used to select data from incoming JSON events
and for inserting data into outgoing JSON events. In addition to event transformation, JPath expressions
also play a key role in matching events to handlers and in filtering of events.

## Simple Structural Transformations

For example, the following excerpt from a transformation descriptor selects everything under `/data` from the
incoming event (right hand side) and inserts it under `/event/data` into the outgoing event (left hand side).

```
{
  "{{/event/data}}" : "{{/data}}"
}
```

Input event:

```
{
  "data" : {
    "status" : "ok",
    "deviceId": 123
  }
}
```

Output event:

```
{
  "event" : {
    "data" : {
      "status" : "ok",
      "deviceId": 123
    }
  }
}
```

JPath expressions can be used to select simple data fields (string, float, boolean) or complex
ones with nested elements (map, array).

## Array Path Selectors

You can select from arrays by index or by key.

Input event:

```
{
  "foo" : [
    { "name" : "status" , "value" : "online" },
    { "name" : "temperature" , "value" : 61 }
  ]
}
```

_Example 1:_

```
{{/foo[0]/value}}
```

Result is `"online"`.

_Example 2:_

```
{{/foo[name=temperature]/value}}
```

Result is `61`.

## Functions

JPath expressions can also include function calls for more complex tasks, such as injecting a
timestamp or injecting JSON results from an external web service call. See the complete function
reference [here](functions.md). Function calls can be nested.

```
{
  "{{/data/timestamp}}" : "{{time()}}"
}
```

Results of function calls can also be concatenated.

```
{
  "{{/data/uniqueId}}" : "{{time()}}-{{uuid()}}"
}
```

## Escape Characters

Use the `$` sign to escape `{{` or `}}` to prevent EEL from interpreting the content inside as a JPath
expression. This is useful when a downstream service requires mustache notation for further processing.

Example:

```
"{{/message}}":"User email is ${{email$}}"
```

Output event:

```
{ "message" : "User email is {{email}}" }
```

To escape individual characters such as single quote inside of a function parameter, use the 
backslash character.

Example:

```
{{ident('this wasn\'t working in earlier versions')}}
```
