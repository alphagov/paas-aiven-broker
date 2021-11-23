package aiven

type CommonUserConfig struct {
	IPFilter          []string `json:"ip_filter,omitempty"`
	ForkProject       string   `json:"project_to_fork_from,omitempty"`
	BackupServiceName string   `json:"service_to_fork_from,omitempty"`
	BackupName        string   `json:"recovery_basebackup_name,omitempty"`
}

type ElasticsearchUserConfig struct {
	ElasticsearchVersion string `json:"elasticsearch_version,omitempty"`
}
type OpenSearchUserConfig struct {
	OpenSearchVersion string `json:"opensearch_version,omitempty"`
}

type InfluxDBUserConfig struct{}

type UserConfig struct {
	CommonUserConfig
	ElasticsearchUserConfig
	OpenSearchUserConfig
	InfluxDBUserConfig
}
