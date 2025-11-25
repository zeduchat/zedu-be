package config

import (
	"log"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"

	"github.com/hngprojects/telex_be/utility"
)

// Setup initialize configuration
var (
	// Params ParamsConfiguration
	Config *Configuration
)

// Params = getConfig.Params
func Setup(logger *utility.Logger, name string) *Configuration {
	var baseConfiguration *BaseConfig

	viper.SetConfigName(name)
	viper.SetConfigType("env")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Error reading config file, %s", err)
		log.Printf("Will attempt to read from environment variables only")
	} else {
		log.Printf("Successfully loaded config from file: %s", viper.ConfigFileUsed())
	}

	viper.AutomaticEnv()

	var config BaseConfig

	err := BindKeys(viper.GetViper(), config)
	if err != nil {
		log.Fatalf("Unable to bind keys in struct, %v", err)
	}

	err = viper.Unmarshal(&baseConfiguration)
	if err != nil {
		log.Fatalf("Unable to decode into struct, %v", err)
	}

	configuration := baseConfiguration.SetupConfigurationn()

	// Params = configuration.Params
	Config = configuration
	logger.Info("configurations loading successfully")
	return configuration
}

// GetConfig helps you to get configuration data
func GetConfig() *Configuration {
	return Config
}

func BindKeys(v *viper.Viper, input any) error {

	envKeysMap := &map[string]any{}
	if err := mapstructure.Decode(input, &envKeysMap); err != nil {
		return err
	}
	for k := range *envKeysMap {
		if bindErr := viper.BindEnv(k); bindErr != nil {
			return bindErr
		}
	}

	return nil
}
