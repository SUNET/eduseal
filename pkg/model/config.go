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

// TLS holds the tls configuration
type TLS struct {
	Enabled      bool   `yaml:"enabled"`
	CertFilePath string `yaml:"cert_file_path"`
	KeyFilePath  string `yaml:"key_file_path"`
	RootCAPath   string `yaml:"root_ca_path"`
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
	SealerNodes          []string `yaml:"sealer_nodes" validate:"omitempty"`
	SealerServiceName    string   `yaml:"sealer_service_name" validate:"omitempty"`
	ValidatorNodes       []string `yaml:"validator_nodes" validate:"omitempty"`
	ValidatorServiceName string   `yaml:"validator_service_name" validate:"omitempty"`
	RootCAPath           string   `yaml:"root_ca_path"`
	Redict               Redict   `yaml:"redict" validate:"required"`
	Queue                Queue    `yaml:"queue" validate:"required"`
}

// Redict holds the key/value configuration
type Redict struct {
	Nodes    []string `yaml:"nodes" validate:"required"`
	Password string   `yaml:"password" validate:"required"`
}

// SMT Spares Merkel Tree configuration
type SMT struct {
	UpdatePeriodicity int    `yaml:"update_periodicity" validate:"required"`
	InitLeaf          string `yaml:"init_leaf" validate:"required"`
}

// GRPCServer holds the rpc configuration
type GRPCServer struct {
	Addr   string `yaml:"addr" validate:"required"`
	Secure bool   `yaml:"secure"`
}

// PDF holds the pdf configuration (special Ladok case)
type PDF struct {
	KeepSignedDuration   int `yaml:"keep_signed_duration"`
	KeepUnsignedDuration int `yaml:"keep_unsigned_duration"`
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
	Type    string `yaml:"type" validate:"required"`
	Timeout int64  `yaml:"timeout" validate:"required"`
}

// Queue holds the queue configuration
type Queue struct {
	Username string   `yaml:"username" validate:"required"`
	Password string   `yaml:"password" validate:"required"`
	Addr     []string `yaml:"addr" validate:"required"`
}

// Cfg is the main configuration structure for this application
type Cfg struct {
	Common Common `yaml:"common"`
	APIGW  APIGW  `yaml:"apigw" validate:"omitempty"`
}
