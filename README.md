# Universal Service Broker

This is a partial implementation of a service broker conforming to the [Open Service Broker API](https://www.openservicebrokerapi.org/). It is intended to save time creating new service brokers by implementing the parts which are generic to all brokers of this kind. It can be used in two ways:

1. Imported as a library
1. Copying the code

## Usage

This code is used as a starting point. Your own code will involve two responsibilities:

1. Implementing a 'provider', which interacts with an API to manage your service.
1. Managing its own configuration and service catalog.

### Implementing the provider

In order to complete your own service broker you need to implement the [`provider` interface](https://github.com/henrytk/universal-service-broker/blob/master/provider/interface.go). These methods cover each of the lifecycle management operations for a service instance.

### Managing configuration and a service catalog

You must provide a JSON file such as [the example configuration](examples/config.json). You can see in the [configuration tests](broker/config_test.go) which fields are mandatory and which have defaults you can use. The `catalog` field must exist and contain a list of services as defined in the [catalog management](https://github.com/openservicebrokerapi/servicebroker/blob/v2.13/spec.md#catalog-management) section of the Open Service Broker API specification.

The `broker` package provides a `NewConfig` method for parsing the configuration file. This config is passed in when creating a new broker with the `New` method. The configuration type contains a `Provider` field which contains the raw configuration file as a slice of bytes. Your broker implementation is responsible for knowing how to unmarshal this data.

### Example

A full example usage can be found in the [aws-service-broker repository](https://github.com/henrytk/aws-service-broker).

## Testing

Tests for this repository are run with:

```
ginkgo -r
```

They are mostly unit tests. Integration with the upstream `brokerapi` library are covered by the [API tests](broker/api_test.go).

### Testing your implementation

Once you have a working broker you can use the helper methods in the `broker/testing` package to test it as a broker client would.