package config

// ServerConfiguration model for server behaviour
type ServerConfiguration struct {
	Mode            string
	Addr            string
	LogDuration     int
	ShutdownTimeout int
}
