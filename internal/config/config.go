package config

type Config struct {
	TmplPath string
}

func NewConfig() Config {

	return Config{
		TmplPath: "config/templates/",
	}

}
