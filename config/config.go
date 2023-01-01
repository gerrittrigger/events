package config

type Config struct {
	ApiVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	MetaData   MetaData `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

type MetaData struct {
	Name string `yaml:"name"`
}

type Spec struct {
	Connect  Connect  `yaml:"connect"`
	Log      Log      `yaml:"log"`
	Queue    Queue    `yaml:"queue"`
	Storage  Storage  `yaml:"storage"`
	Watchdog Watchdog `yaml:"watchdog"`
}

type Connect struct {
	Hostname string `yaml:"hostname"`
	Ssh      Ssh    `yaml:"ssh"`
}

type Log struct {
}

type Queue struct {
}

type Ssh struct {
	Keyfile         string `yaml:"keyfile"`
	KeyfilePassword string `yaml:"keyfilePassword"`
	Port            int    `yaml:"port"`
	Username        string `yaml:"username"`
}

type Storage struct {
	Autoclean string `yaml:"autoclean"`
	Sqlite    Sqlite `yaml:"sqlite"`
}

type Sqlite struct {
	Filename string `yaml:"filename"`
}

type Watchdog struct {
	PeriodSeconds  int `yaml:"periodSeconds"`
	TimeoutSeconds int `yaml:"timeoutSeconds"`
}

var (
	Build   string
	Version string
)

func New() *Config {
	return &Config{}
}
