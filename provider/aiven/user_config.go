package aiven

type CommonUserConfig struct {
	IPFilter []string `json:"ip_filter,omitempty"`
}

type ElasticsearchUserConfig struct {
	ElasticsearchVersion string `json:"elasticsearch_version,omitempty"`
}

type InfluxDBUserConfig struct{}

type UserConfig struct {
	CommonUserConfig
	ElasticsearchUserConfig
	InfluxDBUserConfig
}
