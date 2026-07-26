package app

import (
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/epicoon/lxgo/kernel"
	"github.com/epicoon/lxgo/kernel/cast"

	_ "github.com/lib/pq"
)

// ConnectionConfig configures a Connection - see Connection.SetConfig.
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

/** @interface kernel.IConnection */

// Connection is the default kernel.IConnection implementation - a
// PostgreSQL connection via database/sql.
type Connection struct {
	app kernel.IApp
	cfg *ConnectionConfig
	db  *sql.DB
}

var _ kernel.IConnection = (*Connection)(nil)

/** @constructor */

// NewConnection constructs an unconfigured Connection.
func NewConnection() *Connection {
	return new(Connection)
}

// SetApp binds the connection to its owning app.
func (c *Connection) SetApp(app kernel.IApp) {
	c.app = app
}

// SetConfig converts cfg into a ConnectionConfig.
func (c *Connection) SetConfig(cfg kernel.IDict) {
	c.cfg = new(ConnectionConfig)
	dict, err := cast.To[kernel.Dict](cfg)
	if err != nil {
		return
	}
	cast.DictToStruct(dict, c.cfg)
}

// DB returns the underlying *sql.DB, or nil before Connect succeeds.
func (c *Connection) DB() *sql.DB {
	return c.db
}

// Connect opens the connection, retrying up to ConnectAttempts times
// (10 by default) with ConnectAttemptDelay seconds between attempts (2 by default).
func (c *Connection) Connect() error {
	cfg := c.cfg
	if err := validateConfig(cfg); err != nil {
		return err
	}

	SSLMode := cfg.SSLMode
	if SSLMode == "" {
		SSLMode = "disable"
	}
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, SSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return fmt.Errorf("failed to open database connection: %w", err)
	}

	attempts := cfg.ConnectAttempts
	if attempts == 0 {
		attempts = 10
	}
	attDelay := cfg.ConnectAttemptDelay
	if attDelay == 0 {
		attDelay = 2
	}
	delay := time.Duration(attDelay) * time.Second

	for i := 1; i <= attempts; i++ {
		if err = db.Ping(); err == nil {
			c.db = db
			return nil
		}
		time.Sleep(delay)
	}

	return fmt.Errorf("failed to connect to DB after %d attempts: %w", attempts, err)
}

// Close closes the connection.
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
