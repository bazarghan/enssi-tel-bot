package config

type Config struct {
	Bot      Bot      `mapstructure:"bot"`
	Postgres Postgres `mapstructure:"postgres"`
}

type Bot struct {
	Token string `mapstructure:"token"`
}

type Postgres struct {
	Host     string `mapstructure:"host"`
	Port     string `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}
