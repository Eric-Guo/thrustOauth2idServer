
copy the configuration file to the configs directory and binary file before starting the service.

Build with the updated Sponge checkout selected by `go.mod`; the runtime port
requires changes in both repositories. See [Thruster compatibility](../../docs/thruster-port.md).
For public TLS termination, set `proxy.forwardHeaders: false`. The optional
`http.gzipDisableOnAuth` guard and `http.gzipJitter` padding apply to eligible
responses across the Gin and Rails routes. Supervisors now receive Rails' exit
status, including 128 plus the signal number when Rails dies from a signal.

```
├── configs
│         └── thrustOauth2idServer.yml
├── thrustOauth2idServer
├── deploy.sh
└── run.sh
```

### Running and stopping service manually

Running service:

> ./run.sh

Stopping the service:

> ./run.sh stop

<br>

### Automated deployment service

> ./deploy.sh
