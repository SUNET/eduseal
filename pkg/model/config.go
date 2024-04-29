package model

// APIServer holds the api server configuration
type APIServer struct {
	Addr       string            `yaml:"addr" validate:"required"`
	PublicKeys map[string]string `yaml:"public_keys"`
	TLS        TLS               `yaml:"tls" validate:"omitempty"`
}

// TLS holds the tls configuration
type TLS struct {
	Enabled      bool   `yaml:"enabled"`
	CertFilePath string `yaml:"cert_file_path" validate:"required"`
	KeyFilePath  string `yaml:"key_file_path" validate:"required"`
}

// Mongo holds the database configuration
type Mongo struct {
	URI string `yaml:"uri" validate:"required"`
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
	QR         QRCfg             `yaml:"qr" validate:"omitempty"`
}

// SMT Spares Merkel Tree configuration
type SMT struct {
	UpdatePeriodicity int    `yaml:"update_periodicity" validate:"required"`
	InitLeaf          string `yaml:"init_leaf" validate:"required"`
}

// RPCServer holds the rpc configuration
type RPCServer struct {
	Addr     string `yaml:"addr" validate:"required"`
	Insecure bool   `yaml:"insecure"`
}

// PDF holds the pdf configuration (special Ladok case)
type PDF struct {
	KeepSignedDuration   int `yaml:"keep_signed_duration"`
	KeepUnsignedDuration int `yaml:"keep_unsigned_duration"`
}

// QRCfg holds the qr configuration
type QRCfg struct {
	BaseURL       string `yaml:"base_url" validate:"required"`
	RecoveryLevel int    `yaml:"recovery_level" validate:"required,min=0,max=3"`
	Size          int    `yaml:"size" validate:"required"`
}

// Queues have the queue configuration
type Queues struct {
	SimpleQueue struct {
		EduSealSign struct {
			Name string `yaml:"name" validate:"required"`
		} `yaml:"eduseal_sign" validate:"required"`
		EduSealValidate struct {
			Name string `yaml:"name" validate:"required"`
		} `yaml:"eduseal_validate" validate:"required"`
		EduSealAddSigned struct {
			Name string `yaml:"name" validate:"required"`
		} `yaml:"eduseal_add_signed" validate:"required"`
		EduSealDelSigned struct {
			Name string `yaml:"name" validate:"required"`
		} `yaml:"eduseal_del_signed" validate:"required"`
		EduSealPersistentSave struct {
			Name string `yaml:"name" validate:"required"`
		} `yaml:"eduseal_persistent_save" validate:"required"`
	} `yaml:"simple_queue" validate:"required"`
}

// Issuer holds the issuer configuration
type Issuer struct {
	APIServer APIServer `yaml:"api_server" validate:"required"`
	RPCServer RPCServer `yaml:"rpc_server" validate:"required"`
}

// Registry holds the registry configuration
type Registry struct {
	APIServer APIServer `yaml:"api_server" validate:"required"`
	SMT       SMT       `yaml:"smt" validate:"required"`
	RPCServer RPCServer `yaml:"rpc_server" validate:"required"`
}

// Cache holds the cache storage configuration
type Cache struct {
	APIServer APIServer `yaml:"api_server" validate:"required"`
}

// Persistent holds the persistent storage configuration
type Persistent struct {
	APIServer APIServer `yaml:"api_server" validate:"required"`
}

// MockAS holds the mock as configuration
type MockAS struct {
	APIServer    APIServer `yaml:"api_server" validate:"required"`
	DatastoreURL string    `yaml:"datastore_url" validate:"required"`
}

// Verifier holds the verifier configuration
type Verifier struct {
	APIServer APIServer `yaml:"api_server" validate:"required"`
	RPCServer RPCServer `yaml:"rpc_server" validate:"required"`
}

// Datastore holds the datastore configuration
type Datastore struct {
	APIServer APIServer `yaml:"api_server" validate:"required"`
	RPCServer RPCServer `yaml:"rpc_server" validate:"required"`
}

// APIGW holds the datastore configuration
type APIGW struct {
	APIServer APIServer `yaml:"api_server" validate:"required"`
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
}
