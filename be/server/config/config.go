package config

type Config struct {
	Port                string
	DefaultUsername     string
	DefaultPassword     string
	DefaultPushRegistry string
	DefaultPullRegistry string
	Minio               MinioConfig
}

type MinioConfig struct {
	Endpoint string
	Username string
	Password string
}

func Load() *Config {
	return &Config{
		Port:                "7152",
		DefaultUsername:     "kedong@yunlizhihui",
		DefaultPassword:     "kedong@123",
		DefaultPushRegistry: "172.27.35.4:5000",
		DefaultPullRegistry: "registry.hub.docker.com",
		Minio: MinioConfig{
			Endpoint: "127.0.0.1:9000",
			Username: "admin",
			Password: "minio@Yunli123",
		},
		// Minio: MinioConfig{
		// 	Endpoint: "10.62.210.66:9900",
		// 	Username: "admin",
		// 	Password: "minio@yunli123",
		// },
	}
}
