package app

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/config"
	"github.com/epicoon/lxgo/kernel/conv"

	_ "github.com/lib/pq"
)

type ConnectionConfig struct {
	Host                string
	Port                int
	User                string
	Password            string
	DBName              string
	SSLMode             string
	ConnectAttempts     int
	ConnectAttemptDelay int
}

type Connection struct {
	app kernel.IApp
	cfg *ConnectionConfig
	db  *sql.DB
}

func NewConnection() *Connection {
	return new(Connection)
}

func (c *Connection) SetApp(app kernel.IApp) {
	c.app = app
}

func (c *Connection) SetConfig(cfg *kernel.Config) {
	val, err := config.GetParam[string](cfg, "SSLMode")
	if err != nil || val == "" {
		config.SetParam(cfg, "SSLMode", "disable")
	}
	att, err := config.GetParam[int](cfg, "ConnectAttempts")
	if err != nil || att == 0 {
		config.SetParam(cfg, "ConnectAttempts", 10)
	}
	att, err = config.GetParam[int](cfg, "ConnectAttemptDelay")
	if err != nil || att == 0 {
		config.SetParam(cfg, "ConnectAttemptDelay", 2)
	}
	c.cfg = new(ConnectionConfig)

	conv.DictToStruct((*kernel.Dict)(cfg), c.cfg)
}

func (c *Connection) DB() *sql.DB {
	return c.db
}

// Try to establish connection to DB
func (c *Connection) Connect() error {
	cfg := c.cfg
	if err := validateConfig(cfg); err != nil {
		return err
	}

	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	maxRetries := cfg.ConnectAttempts
	delay := time.Duration(cfg.ConnectAttemptDelay) * time.Second

	for i := 1; i <= maxRetries; i++ {
		if err = db.Ping(); err == nil {
			c.db = db
			return nil
		}
		time.Sleep(delay)
	}

	return fmt.Errorf("failed to connect to DB after %d attempts: %w", maxRetries, err)
}

func (c *Connection) Close() error {
	if c.db != nil {
		return c.db.Close()
	}
	return nil
}

func validateConfig(cfg *ConnectionConfig) error {
	if cfg == nil {
		return errors.New("DB connection config is not defined")
	}

	requiredFields := map[string]string{
		"host":     cfg.Host,
		"port":     fmt.Sprintf("%d", cfg.Port),
		"user":     cfg.User,
		"password": cfg.Password,
		"name":     cfg.DBName,
	}

	for field, value := range requiredFields {
		if value == "" || (field == "port" && value == "0") {
			return fmt.Errorf("undefined db %s", field)
		}
	}

	return nil
}
