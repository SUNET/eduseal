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
}

// TLS holds the tls configuration
type TLS struct {
	Enabled      bool   `yaml:"enabled"`
	CertFilePath string `yaml:"cert_file_path" validate:"required"`
	KeyFilePath  string `yaml:"key_file_path" validate:"required"`
}

// Mongo holds the database configuration
type Mongo struct {
	URI     string `yaml:"uri" validate:"required"`
	Disable bool   `yaml:"disable" validate:"required"`
}

// KeyValue holds the key/value configuration
type KeyValue struct {
	Addr     string `yaml:"addr" validate:"required"`
	DB       int    `yaml:"db" validate:"required"`
	Password string `yaml:"password" validate:"required"`
	PDF      PDF    `yaml:"pdf" validate:"required"`
}

// Log holds the log configuration
type Log struct {
	Level      string `yaml:"level"`
	FolderPath string `yaml:"folder_path"`
}

// Common holds the common configuration
type Common struct {
	HTTPProxy  string            `yaml:"http_proxy"`
	Production bool              `yaml:"production"`
	Log        Log               `yaml:"log"`
	Mongo      Mongo             `yaml:"mongo" validate:"required"`
	BasicAuth  map[string]string `yaml:"basic_auth"`
	Tracing    OTEL              `yaml:"tracing" validate:"required"`
	Queues     Queues            `yaml:"queues" validate:"omitempty"`
	KeyValue   KeyValue          `yaml:"key_value" validate:"omitempty"`
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

// Queues have the queue configuration
type Queues struct {
	SimpleQueue struct {
		EduSealSeal struct {
			Name string `yaml:"name" validate:"required"`
		} `yaml:"eduseal_seal" validate:"required"`
		EduSealValidate struct {
			Name string `yaml:"name" validate:"required"`
		} `yaml:"eduseal_validate" validate:"required"`
		EduSealAddSealed struct {
			Name string `yaml:"name" validate:"required"`
		} `yaml:"eduseal_add_sealed" validate:"required"`
		EduSealDelSealed struct {
			Name string `yaml:"name" validate:"required"`
		} `yaml:"eduseal_del_sealed" validate:"required"`
		EduSealPersistentSave struct {
			Name string `yaml:"name" validate:"required"`
		} `yaml:"eduseal_persistent_save" validate:"required"`
	} `yaml:"simple_queue" validate:"required"`
}

// Cache holds the cache storage configuration
type Cache struct {
	APIServer APIServer `yaml:"api_server" validate:"required"`
}

// Persistent holds the persistent storage configuration
type Persistent struct {
	APIServer APIServer `yaml:"api_server" validate:"required"`
}

// APIGW holds the datastore configuration
type APIGW struct {
	APIServer APIServer `yaml:"api_server" validate:"required"`
	JWTAuth   JWTAuth   `yaml:"jwt_auth" validate:"required"`
}

// Sealer holds the sealer configuration
type Sealer struct {
	GRPCServer GRPCServer `yaml:"grpc_server" validate:"required"`
}

// OTEL holds the opentelemetry configuration
type OTEL struct {
	Addr string `yaml:"addr" validate:"required"`
	Type string `yaml:"type" validate:"required"`
}

// Cfg is the main configuration structure for this application
type Cfg struct {
	Common     Common     `yaml:"common"`
	APIGW      APIGW      `yaml:"apigw" validate:"omitempty"`
	Cache      Cache      `yaml:"cache" validate:"omitempty"`
	Persistent Persistent `yaml:"persistent" validate:"omitempty"`
	Sealer     Sealer     `yaml:"sealer" validate:"omitempty"`
}
