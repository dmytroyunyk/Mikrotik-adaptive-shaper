package config

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultApiPort = 8728

type Config struct {
	RouterOS RouterOSConfig `yaml:"routeros"`
	Agent    AgentConfig    `yaml:"agent"`
	Shaper   ShaperConfig   `yaml:"shaper"`
}

type RouterOSConfig struct {
	Host     string `yaml:"host"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	Port     int    `yaml:"port"`
}

type AgentConfig struct {
	Interval string `yaml:"interval"`
}

func (a AgentConfig) Parsed() (time.Duration, error) {
	return time.ParseDuration(a.Interval)
}

type ShaperConfig struct {
	Interface    string `yaml:"wan_interface"`
	UplinkMbit   int    `yaml:"uplink_mbit"`
	RealtimeMbit int    `yaml:"realtime_mbit"`
	BulkMbit     int    `yaml:"bulk_mbit"`
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("config: cannot open file %s: %w", path, err)
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("config: cannot parse yaml: %w", err)
	}

	cfg.applyDefaults()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil

}

func (c *Config) applyDefaults() {
	if c.RouterOS.Port == 0 {
		c.RouterOS.Port = defaultApiPort
	}
}

func (c *Config) validate() error {
	if c.RouterOS.Host == "" {
		return fmt.Errorf("config: routeros.host is required")
	}
	if c.RouterOS.Username == "" {
		return fmt.Errorf("config: routeros.username is required")
	}
	if c.RouterOS.Password == "" {
		return fmt.Errorf("config: routeros.password is required")
	}

	if c.Agent.Interval == "" {
		return fmt.Errorf("config: agent.interval is required")
	}
	interval, err := c.Agent.Parsed()
	if err != nil {
		return fmt.Errorf("config: agent.interval %q is not a valid duration (e.g. \"1s\", \"500ms\"): %w",
			c.Agent.Interval, err)
	}
	if interval <= 0 {
		return fmt.Errorf("config: agent.interval must be positive, got %s", interval)
	}
	if c.Shaper.Interface == "" {
		return fmt.Errorf("config: shaper.wan_interface is required (e.g. \"ether1\")")
	}
	if c.Shaper.UplinkMbit <= 0 {
		return fmt.Errorf("config: shaper.uplink_mbit is required and must be positive")
	}
	if c.Shaper.RealtimeMbit <= 0 {
		return fmt.Errorf("config: shaper.realtime_mbit is required and must be positive")
	}
	if c.Shaper.BulkMbit <= 0 {
		return fmt.Errorf("config: shaper.bulk_mbit is required and must be positive")
	}

	reserved := c.Shaper.RealtimeMbit + c.Shaper.BulkMbit
	if reserved >= c.Shaper.UplinkMbit {
		return fmt.Errorf(
			"config: realtime_mbit(%d) + bulk_mbit(%d) = %d must be less than uplink_mbit(%d), "+
				"otherwise interactive class gets no guarantee",
			c.Shaper.RealtimeMbit, c.Shaper.BulkMbit, reserved, c.Shaper.UplinkMbit,
		)
	}

	return nil
}
