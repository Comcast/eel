## EEL Web API Reference

### proxy

EEL Web Hook for receiving JSON events for event forwarding:

[http://localhost:8080/v1/proxy](http://localhost:8080/v1/proxy)

### proc

EEL Web API for processing JSON events synchronously and returning the transformed event immediately in the response body:

[http://localhost:8080/v1/proc](http://localhost:8080/v1/proc)

### health

EEL health check. Retrieves JSON encoded status information including current version and configured handlers:

[http://localhost:8080/v1/health](http://localhost:8080/v1/health)

### pluginconfigs

EEL event plugin configuration. Currently only the webhook plugin (enabled by default) and the stdin plugin (disabled by default) are available:

[http://localhost:8080/v1/pluginconfigs](http://localhost:8080/v1/pluginconfigs)

### reload

Reload config.json and handlers from disk without restarting EEL:

[http://localhost:8080/v1/reload](http://localhost:8080/reload)

### vet

Vet all configured handlers and returns list of warnings:

[http://localhost:8080/v1/vet](http://localhost:8080/vet)
