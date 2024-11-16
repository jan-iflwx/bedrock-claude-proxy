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

func (this *Config) MarginWithENV() {
	if len(this.WebRoot) <= 0 {
		this.WebRoot = os.Getenv("WEB_ROOT")
	}
	if len(this.Listen) <= 0 {
		this.Listen = os.Getenv("HTTP_LISTEN")
	}
	if len(this.APIKey) <= 0 {
		this.APIKey = os.Getenv("API_KEY")
	}
	if this.BedrockConfig == nil {
		this.BedrockConfig = LoadBedrockConfigWithEnv()
	} else {
		envBedrockConfig := LoadBedrockConfigWithEnv()

		if envBedrockConfig.AccessKey != "" {
			this.BedrockConfig.AccessKey = envBedrockConfig.AccessKey
		}
		if envBedrockConfig.SecretKey != "" {
			this.BedrockConfig.SecretKey = envBedrockConfig.SecretKey
		}
		if envBedrockConfig.Region != "" {
			this.BedrockConfig.Region = envBedrockConfig.Region
		}
		if envBedrockConfig.RoleArn != "" {
			this.BedrockConfig.RoleArn = envBedrockConfig.RoleArn
		}
		if envBedrockConfig.RoleRegion != "" {
			this.BedrockConfig.RoleRegion = envBedrockConfig.RoleRegion
		}
		if len(envBedrockConfig.ModelMappings) > 0 {
			this.BedrockConfig.ModelMappings = envBedrockConfig.ModelMappings
		}
		if len(envBedrockConfig.AnthropicVersionMappings) > 0 {
			this.BedrockConfig.AnthropicVersionMappings = envBedrockConfig.AnthropicVersionMappings
		}
		if envBedrockConfig.AnthropicDefaultModel != "" {
			this.BedrockConfig.AnthropicDefaultModel = envBedrockConfig.AnthropicDefaultModel
		}
		if envBedrockConfig.AnthropicDefaultVersion != "" {
			this.BedrockConfig.AnthropicDefaultVersion = envBedrockConfig.AnthropicDefaultVersion
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

func (c *Config) ToJSON() (string, error) {
	jsonBin, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	var str bytes.Buffer
	_ = json.Indent(&str, jsonBin, "", "  ")
	return str.String(), nil
}

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
