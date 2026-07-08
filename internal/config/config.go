package config

type Config struct {
	App     AppConfig     `yaml:"app"`
	Server  ServerConfig  `yaml:"server"`
	Edge    EdgeConfig    `yaml:"edge"`
	Core    CoreConfig    `yaml:"core"`
	Logging LoggingConfig `yaml:"logging"`
	Storage StorageConfig `yaml:"storage"`
}

type AppConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type ServerConfig struct {
	HTTPPort int `yaml:"http_port"`
	GRPCPort int `yaml:"grpc_port"`
}

type EdgeConfig struct {
	Name            string `yaml:"name"`
	Host            string `yaml:"host"`
	DisableAutoSync bool   `yaml:"disable_auto_sync"`
}

type CoreConfig struct {
	Host     string `yaml:"host"`
	HTTPPort int    `yaml:"http_port"`
	GRPCPort int    `yaml:"grpc_port"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
}

type StorageConfig struct {
	DataDir   string `yaml:"data_dir"`
	LogDir    string `yaml:"log_dir"`
	MountPath string `yaml:"mount_path"`
}
