package broker

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/pivotal-cf/brokerapi"
)

type Config struct {
	BasicAuthUsername string                    `json:"basic_auth_username"`
	BasicAuthPassword string                    `json:"basic_auth_password"`
	Catalog           brokerapi.CatalogResponse `json:"catalog"`
}

func NewConfig(source io.Reader) (Config, error) {
	config := Config{}
	bytes, err := ioutil.ReadAll(source)
	if err != nil {
		return config, err
	}
	return config, json.Unmarshal(bytes, &config)
}

func (c Config) Validate() error {
	if c.BasicAuthUsername == "" {
		return fmt.Errorf("Config error: basic auth username required")
	}
	if c.BasicAuthPassword == "" {
		return fmt.Errorf("Config error: basic auth password required")
	}
	return nil
}
