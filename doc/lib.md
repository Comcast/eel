# Using EEL as Library

You can perform EEL transformations on arbitrary JSON content in go code by using the `EELTransformEvent()` API.

```
// initialize context

ctx := NewDefaultContext(L_InfoLevel)
EELInit(ctx)

// load handlers from folder: note parameter is an array of one or more folders

eelHandlerFactory, warnings := EELNewHandlerFactory(ctx, "./config-handlers")

// check if parsing handlers caused warnings

for _, w := range warnings {
	fmt.Printf("warning loading handlers: %s\n", w)
}

// prepare incoming test event, event should be of type interface{} or []interface{} or map[string]interface{}

in := map[string]interface{}{
	"message": "hello world!!!",
}

// process event and get publisher objects in return - typically we expect exactly one publisher (unless event was filtered)

outs, err := EELTransformEvent(ctx, in, eelHandlerFactory)

// assuming we only have the default handler with identity transformation, the following asserions should hold

if err != nil {
	fmt.Printf("could not transform event: %s\n", err.Error())
}

if len(outs) != 1 {
	fmt.Printf("unexpected number of results: %d\n", len(outs))
}


if !DeepEquals(outs[0], in) {
	fmt.Printf("unexpected transformation result")
}

// if you wish to use the EEL library to forward events to endpoints use EELGetPublishers() instead of EELTransformEvent()

//publishers, err := EELGetPublishers(ctx, in, eelHandlerFactory)
//
//if err != nil {
//	fmt.Printf("could not transform event: %s\n", err.Error())
//}
//
//for _, p := range publishers {
//	resp, err := p.Publish()
//	if err != nil {
//		fmt.Printf("could not publish event: %s\n", err.Error())
//	}
//	fmt.Printf("response: %s\n", resp)
//}
```

Use the `EELSingleTransform()` API for an easy way to execute simple transformations.  

```
// initialize context

ctx := NewDefaultContext(L_InfoLevel)
EELInit(ctx)

// prepare event and transformation

event := `{ "message" : "hello world!!!" }`
transformation := `{ "{{/event}}" : "{{/}}" }`

// perform trasnformation

out, err := EELSingleTransform(ctx, event, transformation, false)

if err != nil {
	fmt.Printf("bad tranformation: %s\n", err.Error())
}

fmt.Printf("transformed event: %s\n", out)
```
