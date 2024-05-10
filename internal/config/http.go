package config

type HTTP struct {
	Port           string `json:"port"`
	RequestTimeout int    `json:"requestTimeout"`
}

func httpConfig() HTTP {
	return HTTP{
		Port:           getEnv("HTTP_PORT"),
		RequestTimeout: getEnvAsInt("HTTP_REQUEST_TIMEOUT"),
	}
}
