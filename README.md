# Aiven Service Broker

⚠️
When merging pull requests,
please use the [gds-cli](https://github.com/alphagov/gds-cli)
or [github_merge_sign](https://rubygems.org/gems/github_merge_sign)
⚠️

This is an [Open Service Broker API](https://www.openservicebrokerapi.org/) compliant service broker for services offered by [https://aiven.io/](https://aiven.io/).

## Running

Provide a configuration file, such as the example configuration, and run `main`:

```bash
go run main.go -config examples/config.json
```

## Testing

### Unit Testing

```bash
make unit
```

### Integration Testing

Integration testing requires environment variables to be set (see [`provider/config.go`](https://github.com/alphagov/paas-aiven-broker/blob/main/provider/config.go#L70-L90) for details).

There is a make target which creates a `.env` file with these credentials and runs with these:

```bash
make local_integration
```

Alternatively, export these manually and run:

```bash
make integration
```

Note: integration testing uses the real Aiven API and therefore incurs a cost.

<!-- 2020-12-07[T]11:00:00 -->
