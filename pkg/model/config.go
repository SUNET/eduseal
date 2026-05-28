package model

// APIServer holds the api server configuration
type APIServer struct {
	Addr       string            `yaml:"addr" validate:"required"`
	PublicKeys map[string]string `yaml:"public_keys"`
	TLS        TLS               `yaml:"tls" validate:"omitempty"`
}

// JWTAuth holds the jwt auth configuration
type JWTAuth struct {
	Enabled bool              `yaml:"enabled"`
	Access  map[string]string `yaml:"access"`
	JWKURL  string            `yaml:"jwk_url"`
}

// TLS holds the tls configuration.
//
// When Enabled is true, CertFilePath and KeyFilePath are required.
// RootCAPath is optional and only used when verifying a peer against a
// custom CA (client TLS / mTLS).
// ServerName overrides the server name used for certificate verification.
// It is only used by the queue (NATS) connection.  KV backends (Valkey /
// Redict) derive the correct ServerName per-connection from the configured
// node addresses, so ServerName is not needed for KV.
type TLS struct {
	Enabled      bool   `yaml:"enabled"`
	CertFilePath string `yaml:"cert_file_path" validate:"required_if=Enabled true"`
	KeyFilePath  string `yaml:"key_file_path" validate:"required_if=Enabled true"`
	RootCAPath   string `yaml:"root_ca_path"`
	ServerName   string `yaml:"server_name"`
}

// Mongo holds the database configuration
type Mongo struct {
	URI     string `yaml:"uri" validate:"required"`
	Disable bool   `yaml:"disable" validate:"required"`
}

// Log holds the log configuration
type Log struct {
	Level      string `yaml:"level"`
	FolderPath string `yaml:"folder_path"`
}

// Common holds the common configuration
type Common struct {
	HTTPProxy            string   `yaml:"http_proxy"`
	Production           bool     `yaml:"production"`
	Log                  Log      `yaml:"log"`
	Mongo                Mongo    `yaml:"mongo" validate:"required"`
	Tracing              OTEL     `yaml:"tracing" validate:"required"`
	Metric               OTEL     `yaml:"metric" validate:"required"`
	ValidatorNodes       []string `yaml:"validator_nodes" validate:"omitempty"`
	ValidatorServiceName string   `yaml:"validator_service_name" validate:"omitempty"`
	RootCAPath           string   `yaml:"root_ca_path"`
	KV                   KV       `yaml:"kv" validate:"required"`
	Queue                Queue    `yaml:"queue" validate:"required"`
}

// KV holds the key/value store configuration.
// Type selects the backend: "valkey" (default) or "redict".
type KV struct {
	Type     string   `yaml:"type" validate:"omitempty,oneof=valkey redict"`
	Nodes    []string `yaml:"nodes" validate:"required"`
	Password string   `yaml:"password"`
	TLS      TLS      `yaml:"tls" validate:"omitempty"`
}

// APIGW holds the datastore configuration
type APIGW struct {
	APIServer  APIServer `yaml:"api_server" validate:"required"`
	JWTAuth    JWTAuth   `yaml:"jwt_auth" validate:"required"`
	ClientCert TLS       `yaml:"client_cert" validate:"required"`
}

// Sealer holds the sealer configuration
//type Sealer struct {
//	GRPCServer GRPCServer `yaml:"grpc_server" validate:"required"`
//}

// OTEL holds the opentelemetry configuration
type OTEL struct {
	Addr    string `yaml:"addr" validate:"required"`
	Timeout int64  `yaml:"timeout" validate:"required"`
}

// Queue holds the queue configuration
type Queue struct {
	Username string   `yaml:"username" validate:"required"`
	Password string   `yaml:"password" validate:"required"`
	Addr     []string `yaml:"addr" validate:"required"`
	TLS      TLS      `yaml:"tls" validate:"omitempty"`
}

// Cfg is the main configuration structure for this application
type Cfg struct {
	Common Common `yaml:"common"`
	APIGW  APIGW  `yaml:"apigw" validate:"omitempty"`
}
