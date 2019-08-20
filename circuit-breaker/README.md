# Circuit Breaker

When there is a failure in the connection to a upstream service
reply with a default response.

Start an async process to check the health of the upstream service.

## Usage

### Starting the breaker

```
go run breaker/main.go
```

### Starting upstream service

```
go run service1/main.go
```

### Break circuit

Hitting the breaker will cause a request to be sent to the upstream service.

```
curl localhost:8080/ping
```

This service will have a bad response and cause the breaker to start an
async process to check it's health at regular intervals.
