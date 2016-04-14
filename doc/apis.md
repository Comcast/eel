## EEL Web API Reference

### /proxy

EEL Web Hook for receiving JSON events for event forwarding:

[http://localhost:8080/proxy](http://localhost:8080/proxy)

### /proc

EEL Web API for processing JSON events synchronously and returning the transformed event immediately in the response body.

[http://localhost:8080/proc](http://localhost:8080/proc)

### /health

EEL health check. Retrieves JSON encoded status information including current version and configured handlers:

[http://localhost:8080/health](http://localhost:8080/health)

### /reload

Reload config.json and handlers from disk without restarting EEL:

[http://localhost:8080/reload](http://localhost:8080/reload)

### /vet

Vet all configured handlers and returns list of warnings:

[http://localhost:8080/vet](http://localhost:8080/vet)
