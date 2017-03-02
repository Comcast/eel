## EEL Configuration

Most global configuration settings can be found in [../config-eel/config.json](../config-eel/config.json).

```
{
    "Name": "Default",
    "EventPort": 8080,
    "EventProxyPath": "/proxy",
    "EventProcPath": "/proc",
    "Endpoint": "http://localhost:9090",
    "MaxMessageSize": 51200,
    "MaxAttempts": 1,
    "InitialDelay": 125,
    "HttpTransactionHeader": "X-B3-TraceId",
    "HttpTimeout": 1000,
    "ResponseHeaderTimeout" : 5000,
    "MaxIdleConnsPerHost":100,
    "WorkerPoolSize": 1,
    "MessageQueueTimeout": 1000,
    "MessageQueueDepth": 100,
    "LogStats": "false"
    "DuplicateTimeout": 20000,
    "CustomProperties": {
        "key" : "value"
    },
}
```

Most of the settings are self-explanatory. In general, if a setting is blank or missing the feature it configures
will be disabled.

* `Name` - EEL deployment name. Primarily used for logging.
* `EventPort`, `EventProxyPath`, `EventProcPath` - Endpoint where EEL is listening for incoming events. Default is `http://localhost:8080/proxy` for event forwarding and `http://localhost:8080/proc` for synchronous event processing.
* `Endpoint` - Default endpoint for downstream service. Can be a flat string or an array of multiple endpoints. Can be overwritten by `Endpoint` in handler configuration.
* `MaxMessageSize` - Maximum message size EEL will accept from upstream service.
* `MaxAttempts` - If forwarding a message fails, this is the number of attempts EEL will retry with exponential backoff.
* `InitialDelay` - Initial delay for exponential backoff algorithm.
* `HttpTransactionHeader` - Zipkin compliant HTTP transaction ID header. The value for this header has to be configured in each handler separately.
* `WorkerPoolSize`, `MessageQueueTimeout`, `MessageQueueDepth` - Worker pool settings for EEL event handling.
* `MaxIdleConnsPerHost`, `HttpTimeout`, `ResponseHeaderTimeout` - Http settings for outgoing events.
* `LogStats` - Boolean to turn stats logging (typically once a minute) on or off.
* `DuplicateTimeout` - If > 0 will de-duplicated events with a TTL of `DuplicateTimeout` ms.
* `CustomProperties` - Custom properties, can be accessed using the `{{prop('key')}}` function.

Plugins for consuming events from different event sources are configured in [../config-eel/plugins.json](../config-eel/plugins.json).
By default EEL comes with a web hook plugin and a stdin plugin but it is easy to provide your own plugin for any other source of JSON events.

```
[
	{
		"Type" : "WEBHOOK",
		"Name" : "WEBHOOK",
		"Active" : true,
		"RestartOk": false,
		"Parameters" : {
		    "EventPort": 8080,
		    "EventProxyPath": "/v1/proxy",
		    "EventProcPath": "/v1/proc"
		}
	},
	{
		"Type" : "STDIN",
		"Name" : "STDIN",
		"Active" : false,
		"RestartOk": false,
		"Parameters" : {}
	}
]
```
