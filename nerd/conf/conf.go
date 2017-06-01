//Package conf gives the CLI access to the nerd config file. By default this config file is
//~/.nerd/config.json, but the location can be changed using SetLocation().
//
//All read and write operation to the config file should go through the Read() and Write() functions.
//This way we can keep an in-memory representation of the config (in the global conf variable) for fast read.
package conf

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
)

//Config is the structure that describes how the config file looks.
type Config struct {
	Auth            AuthConfig    `json:"auth"`
	Logging         LoggingConfig `json:"logging"`
	NerdAPIEndpoint string        `json:"nerd_api_endpoint"`
}

//AuthConfig contains config details with respect to the authentication server.
type AuthConfig struct {
	APIEndpoint      string `json:"api_endpoint"`
	PublicKey        string `json:"public_key"`
	ClientID         string `json:"client_id"`
	OAuthSuccessURL  string `json:"oauth_success_url"`
	OAuthLocalServer string `json:"oauth_localserver"`
}

//LoggingConfig contains config details about local logging of command output
type LoggingConfig struct {
	Enabled      bool   `json:"enabled"`
	FileLocation string `json:"file_location"`
}

//Defaults provides the default for when the config file misses certain fields.
func Defaults() *Config {
	return &Config{
		Auth: AuthConfig{
			APIEndpoint:      "http://auth.nerdalize.com",
			OAuthLocalServer: "localhost:9876",
			OAuthSuccessURL:  "https://cloud.nerdalize.com",
			ClientID:         "GuoeRJLYOXzVa9ydPjKi83lCctWtXpNHuiy46Yux",
			PublicKey: `-----BEGIN PUBLIC KEY-----
MHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEAkYbLnam4wo+heLlTZEeh1ZWsfruz9nk
kyvc4LwKZ8pez5KYY76H1ox+AfUlWOEq+bExypcFfEIrJkf/JXa7jpzkOWBDF9Sa
OWbQHMK+vvUXieCJvCc9Vj084ABwLBgX
-----END PUBLIC KEY-----`,
		},
		Logging: LoggingConfig{
			Enabled:      false,
			FileLocation: "~/.nerd/log",
		},
		NerdAPIEndpoint: "https://batch.nerdalize.com/v1",
	}
}

//GetDefaultConfigLocation sets the location to ~/.nerd/config.json
func GetDefaultConfigLocation() (string, error) {
	dir, err := homedir.Dir()
	if err != nil {
		return "", errors.Wrap(err, "failed to find home dir")
	}
	return filepath.Join(dir, ".nerd", "config.json"), nil
}

//Read reads the config file
func Read(location string) (*Config, error) {
	content, err := ioutil.ReadFile(location)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open config file")
	}
	conf := Defaults()
	err = json.Unmarshal(content, conf)
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse config file")
	}
	return conf, nil
}

//FromJSON returns a Config object from a JSON string
func FromJSON(in string) (*Config, error) {
	v := Defaults()
	return v, json.Unmarshal([]byte(in), v)
}
