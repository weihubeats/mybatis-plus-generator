package configs

type Config struct {
	TmplPath string
}

func NewConfig() Config {

	return Config{
		TmplPath: "configs/tmpl/",
	}

}
