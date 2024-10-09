package redis

type Config struct {
	addresses []string
	username  string
	password  string
}

func NewConfig() *Config {
	return &Config{}
}

func (c *Config) SetAddresses(addresses []string) *Config {
	c.addresses = addresses
	return c
}

func (c *Config) SetUsername(username string) *Config {
	c.username = username
	return c
}

func (c *Config) SetPassword(password string) *Config {
	c.password = password
	return c
}

func (c *Config) Addresses() []string {
	cloned := make([]string, len(c.addresses))
	copy(cloned, c.addresses)
	return cloned
}

func (c *Config) Username() string {
	return c.username
}

func (c *Config) Password() string {
	return c.password
}
