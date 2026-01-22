package config

import (
	"sync"

	"github.com/spf13/viper"
)

type Config struct {
	DBDriver   string `mapstructure:"DB_DRIVER"`
	DBSource   string `mapstructure:"DB_SOURCE"`
	ServerPort string `mapstructure:"SERVER_PORT"`
	JwtPriKey  string `mapstructure:"JWT_PRIVATE_KEY"`
}

var (
	config Config
	once   sync.Once
)

func init() {
	LoadConfig(".")
}

func LoadConfig(path string) {
	once.Do(func() {
		viper.AddConfigPath(path)
		viper.SetConfigName("app")
		viper.SetConfigType("env")
		viper.AutomaticEnv()
		err := viper.ReadInConfig()
		if err != nil {
			panic(err)
		}
		err = viper.Unmarshal(&config)
		if err != nil {
			panic(err)
		}
	})
}

func GetConfig() Config {
	if config == (Config{}) {
		panic("config is not loaded")
	}
	return config
}
