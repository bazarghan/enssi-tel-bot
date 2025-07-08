package config

// Config is the top-level struct holding all application configuration.
type Config struct {
	Database Database `mapstructure:"database"`
	Telegram Telegram `mapstructure:"telegram"`
	Log      Log      `mapstructure:"log"`
}

// Database holds all configuration for the database connection.
type Database struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	Name     string `mapstructure:"name"`
	SSLMode  string `mapstructure:"sslmode"`
}

// Telegram holds all configuration related to the Telegram Bot API.
type Telegram struct {
	Token           string `mapstructure:"token"`
	AdminTelegramID int64  `mapstructure:"admin_telegram_id"`
}

type Log struct {
	Level string `mapstructure:"level"`
}
