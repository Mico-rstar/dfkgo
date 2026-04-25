package config

import (
	"sync"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort            string `mapstructure:"SERVER_PORT"`
	JwtPriKey             string `mapstructure:"JWT_PRIVATE_KEY"`
	JwtDurationHours      int    `mapstructure:"JWT_DURATION_HOURS"`
	DBDriver              string `mapstructure:"DB_DRIVER"`
	DBSource              string `mapstructure:"DB_SOURCE"`
	OSSRegion             string `mapstructure:"OSS_REGION"`
	OSSEndpoint           string `mapstructure:"OSS_ENDPOINT"`
	OSSBucketFiles        string `mapstructure:"OSS_BUCKET_FILES"`
	OSSBucketAvatars      string `mapstructure:"OSS_BUCKET_AVATARS"`
	OSSAccessKeyID        string `mapstructure:"OSS_ACCESS_KEY_ID"`
	OSSAccessKeySecret    string `mapstructure:"OSS_ACCESS_KEY_SECRET"`
	OSSStsRoleArn         string `mapstructure:"OSS_STS_ROLE_ARN"`
	OSSStsDurationSeconds int    `mapstructure:"OSS_STS_DURATION_SECONDS"`
	ModelServerBaseURL    string `mapstructure:"MODEL_SERVER_BASE_URL"`
	ModelServerTimeoutSec int    `mapstructure:"MODEL_SERVER_TIMEOUT_SECONDS"`
	TaskWorkerPoolSize    int    `mapstructure:"TASK_WORKER_POOL_SIZE"`
	TaskQueueCapacity     int    `mapstructure:"TASK_QUEUE_CAPACITY"`
}

var (
	config Config
	once   sync.Once
	loaded bool
)

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
		loaded = true
	})
}

func GetConfig() Config {
	if !loaded {
		LoadConfig(".")
	}
	return config
}
