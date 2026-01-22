package config

import "github.com/spf13/viper"

type Config struct {
	DBDriver   string `mapstructure:"DB_DRIVER"`
	DBSource   string `mapstructure:"DB_SOURCE"`
	ServerPort string `mapstructure:"SERVER_PORT"`
	JwtPriKey  string `mapstructure:"JWT_PRIVATE_KEY"`
}

var config Config

func init() {
	viper.AddConfigPath("..")
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
}


func GetConfig() Config {
	return config
}