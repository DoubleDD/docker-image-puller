package config

type Config struct {
	Port                string
	DefaultUsername     string
	DefaultPassword     string
	DefaultPushRegistry string
	DefaultPullRegistry string
}

func Load() *Config {
	return &Config{
		Port:                "7152",
		DefaultUsername:     "kedong@yunlizhihui",
		DefaultPassword:     "kedong@123",
		DefaultPushRegistry: "172.27.35.4:5000",
		DefaultPullRegistry: "registry.hub.docker.com",
	}
}
