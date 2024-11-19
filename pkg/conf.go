package pkg

import (
	"bytes"
	"encoding/json"
	"os"
)

type Config struct {
	HttpConfig
	BedrockConfig *BedrockConfig `json:"bedrock_config,omitempty"`
}

func NewConfigFromLocal(filename string) (*Config, error) {
	conf := &Config{}
	err := conf.load(filename)
	return conf, err
}

func (config *Config) MarginWithENV() {
	webRoot := os.Getenv("WEB_ROOT")
	if len(webRoot) > 0 {
		config.WebRoot = webRoot
	}

	httpListen := os.Getenv("HTTP_LISTEN")
	if len(httpListen) > 0 {
		config.Listen = httpListen
	}

	apiKey := os.Getenv("API_KEY")
	if len(apiKey) > 0 {
		config.APIKey = apiKey
	}

	envBedrockConfig := LoadBedrockConfigWithEnv()
	if config.BedrockConfig == nil {
		config.BedrockConfig = envBedrockConfig
	} else {
		if envBedrockConfig.AccessKey != "" {
			config.BedrockConfig.AccessKey = envBedrockConfig.AccessKey
		}
		if envBedrockConfig.SecretKey != "" {
			config.BedrockConfig.SecretKey = envBedrockConfig.SecretKey
		}
		if envBedrockConfig.Region != "" {
			config.BedrockConfig.Region = envBedrockConfig.Region
		}
		if envBedrockConfig.RoleArn != "" {
			config.BedrockConfig.RoleArn = envBedrockConfig.RoleArn
		}
		if envBedrockConfig.RoleRegion != "" {
			config.BedrockConfig.RoleRegion = envBedrockConfig.RoleRegion
		}
		if len(envBedrockConfig.ModelMappings) > 0 {
			config.BedrockConfig.ModelMappings = envBedrockConfig.ModelMappings
		}
		if len(envBedrockConfig.AnthropicVersionMappings) > 0 {
			config.BedrockConfig.AnthropicVersionMappings = envBedrockConfig.AnthropicVersionMappings
		}
		if envBedrockConfig.AnthropicDefaultModel != "" {
			config.BedrockConfig.AnthropicDefaultModel = envBedrockConfig.AnthropicDefaultModel
		}
		if envBedrockConfig.AnthropicDefaultVersion != "" {
			config.BedrockConfig.AnthropicDefaultVersion = envBedrockConfig.AnthropicDefaultVersion
		}
	}
}

func (c *Config) load(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		Log.Error(err)
		return err
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	err = decoder.Decode(c)
	if err != nil {
		Log.Error(err)
	}
	return err
}

// used for debug only
func (c *Config) ToJSON() (string, error) {
	jsonBin, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	var str bytes.Buffer
	_ = json.Indent(&str, jsonBin, "", "  ")
	return str.String(), nil
}

// unused
func (c *Config) Save(saveAs string) error {
	file, err := os.Create(saveAs)
	if err != nil {
		Log.Error(err)
		return err
	}
	defer file.Close()
	data, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		Log.Error(err)
		return err
	}
	_, err = file.Write(data)
	if err != nil {
		Log.Error(err)
	}
	return err
}
