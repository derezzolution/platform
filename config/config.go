package config

import (
	"fmt"
	"log"

	"github.com/jmoiron/jsonq"

	"encoding/json"
	"os"
	"strings"
)

type Configurer interface {
	// Load reads in a configuration as a function of GO_ENV and the validates it. For GO_ENV=development, we'll load
	// the production configuration first and then layer development on top as overrides.
	Load() error

	// LogSummary prints out details about the configuration.
	LogSummary()

	// ReadProperty lets properties be read from the config using a query path
	// (e.g. "objectA.arrayB.0"). If any errors occur, we return an empty string.
	ReadProperty(queryPath string) (string, error)

	// Validates the configuration.
	Validate() error
}

type Config struct {
	Configurer     `json:"-"`
	Env            string `json:"env"`
	LogFile        string `json:"logFile"`
	VerboseLogging bool   `json:"verboseLogging"`
}

func (c *Config) Load() error {
	return LoadConfig(c)
}

func (c *Config) LogSummary() {
	log.Printf("platform configuration summary")
	log.Printf(" environment: ...... %v", c.Env)
	log.Printf(" verbose logging: .. %v", c.VerboseLogging)
}

func (c *Config) ReadProperty(queryPath string) (string, error) {
	return ReadConfigProperty(c, queryPath)
}

func (c *Config) Validate() error {
	return nil
}

func LoadConfig(c Configurer) error {
	// Grab the environmental variable
	env := os.Getenv("GO_ENV")
	if len(env) < 1 {
		env = "development"
	}

	// For development, layer in prod first (so dev can just be overrides).
	if env == "development" {
		err := decodeJson("config-production.json", c)
		if err != nil {
			return err
		}
	}

	err := decodeJson(fmt.Sprintf("config-%s.json", env), c)
	if err != nil {
		return err
	}

	// Validate the config
	err = c.Validate()
	if err != nil {
		return err
	}

	return err
}

func ReadConfigProperty(c Configurer, queryPath string) (string, error) {
	data := map[string]interface{}{}

	// Convert the current config struct into a format jsonq can query
	b, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	decoder := json.NewDecoder(strings.NewReader(string(b)))
	decoder.Decode(&data)

	// Break the query path into a string array that jsonq can work with
	jq := jsonq.NewQuery(data)

	// We only support int, float, bool, and string
	var jqErr error
	intValue, jqErr := jq.Int(strings.Split(queryPath, ".")...)
	if jqErr == nil {
		return fmt.Sprintf("%d", intValue), nil
	}
	floatValue, jqErr := jq.Float(strings.Split(queryPath, ".")...)
	if jqErr == nil {
		return fmt.Sprintf("%f", floatValue), nil
	}
	boolValue, jqErr := jq.Bool(strings.Split(queryPath, ".")...)
	if jqErr == nil {
		return fmt.Sprintf("%t", boolValue), nil // "true" or "false"
	}
	stringValue, jqErr := jq.String(strings.Split(queryPath, ".")...)
	if jqErr == nil {
		return stringValue, nil
	}

	return "", fmt.Errorf("unable to find value for %s (are you sure it exists and is bool, float, int, or string?): "+
		"%s", queryPath, jqErr)
}

func decodeJson(filename string, s interface{}) error {
	// Open the file
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	// Decode the json using given interface
	decoder := json.NewDecoder(file)
	err = decoder.Decode(s)
	if err != nil {
		return err
	}

	return nil
}
