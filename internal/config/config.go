package config

import (
	"github.com/caarlos0/env/v11"
)

type Config struct {
	// LogLevel is the level of logs to output (debug|info|warn|error)
	LogLevel string `env:"LOG_LEVEL" default:"info"`

	// AccountCreationEnabled specifies whether account creation is enabled
	AccountCreationEnabled bool `env:"ACCOUNT_CREATION_ENABLED" default:"true"`

	// NATSClientPrefix is the prefix to use for the NATS client connection (prefix + hostname)
	NATSClientPrefix string `env:"NATS_CLIENT_PREFIX" default:""`

	// NATSURL is the URL (with port) of the NATS server
	NATSURL string `env:"NATS_URL" default:"nats://localhost:4222"`

	// NATSOutgoingBufferSize is the size of the outgoing buffer for NATS connections
	NATSOutgoingBufferSize int `env:"NATS_OUTGOING_BUFFER_SIZE" default:"8388608"` // 8MB

	// ServerPort is the port the server will listen on
	ServerPort int `env:"SERVER_PORT" default:"54231"`

	// MaxServerConnections is the maximum number of concurrent connections the server will accept
	MaxServerConnections int `env:"MAX_SERVER_CONNECTIONS" default:"1000"`

	// ShutdownTimeoutSeconds is the number of seconds to wait for graceful shutdown
	ShutdownTimeoutSeconds int `env:"SHUTDOWN_TIMEOUT_SECONDS" default:"15"`

	// ServerReadTimeoutSeconds is the number of seconds before a read from a client times out
	ServerReadTimeoutSeconds int `env:"SERVER_READ_TIMEOUT_SECONDS" default:"1800"`

	// ServerTLSCertPath is the path to the TLS certificate for the server
	ServerTLSCertPath string `env:"SERVER_TLS_CERT_PATH" default:""`

	// ServerTLSKeyPath is the path to the TLS key for the server
	ServerTLSKeyPath string `env:"SERVER_TLS_KEY_PATH" default:""`

	// DBConnectionString is the Postgres connection string for the database
	DBConnectionString string `env:"DB_CONNECTION_STRING" default:"root:password@tcp(localhost:3306)/mydb?parseTime=true"`

	// DBQueryLogLevel is the level of logs to output for all database queries (debug|info)
	DBQueryLogLevel string `env:"DB_QUERY_LOG_LEVEL" default:"debug"`

	// PasswordSalt is the salt value used for password hashing
	PasswordSalt string `env:"PASSWORD_SALT" default:"somesaltvalue"`

	// MinUsernameLength is the minimum length for usernames
	MinUsernameLength int `env:"MIN_USERNAME_LENGTH" default:"3"`

	// MinPasswordLength is the minimum length for user passwords
	MinPasswordLength int `env:"MIN_PASSWORD_LENGTH" default:"6"`

	// WorldName is the name of the world/server (as displayed to the client)
	// Can only be up to 16 characters long
	WorldName string `env:"WORLD_NAME" default:"GoFFXI"`

	// MaxContentIDsPerAccount is the maximum number of content IDs a player can have
	MaxContentIDsPerAccount int `env:"MAX_CONTENT_IDS_PER_ACCOUNT" default:"3"`

	// XILoaderVersion is the version of the XI Loader to use
	XILoaderVersion string `env:"XI_LOADER_VERSION" default:"2.0.0"`

	// XIClientEnforceVersion specifies whether to enforce the client version
	// 0 = no enforcement
	// 1 = exact version match
	// 2 = version must be greater than or equal to
	XILoaderEnforceVersion int `env:"XI_LOADER_ENFORCE_VERSION" default:"2"`

	// RiseOfZilartEnabled specifies whether the Rise of Zilart expansion is enabled
	RiseOfZilartEnabled bool `env:"RISE_OF_ZILART_ENABLED" default:"true"`

	// ChainsOfPromathiaEnabled specifies whether the Chains of Promathia expansion is enabled
	ChainsOfPromathiaEnabled bool `env:"CHAINS_OF_PROMATHIA_ENABLED" default:"true"`

	// TreasuresOfAhtUrhganEnabled specifies whether the Treasures of Aht Urhgan expansion is enabled
	TreasuresOfAhtUrhganEnabled bool `env:"TREASURES_OF_AHT_URHGAN_ENABLED" default:"true"`

	// WingsOfTheGoddessEnabled specifies whether the Wings of the Goddess expansion is enabled
	WingsOfTheGoddessEnabled bool `env:"WINGS_OF_THE_GODDESS_ENABLED" default:"false"`

	// ACrystallineProphecyEnabled specifies whether the A Crystalline Prophecy expansion is enabled
	ACrystallineProphecyEnabled bool `env:"A_CRYSTALLINE_PROPHECY_ENABLED" default:"false"`

	// AMoogleKupoDEtatEnabled specifies whether the A Moogle Kupo d'Etat expansion is enabled
	AMoogleKupoDEtatEnabled bool `env:"A_MOOGLE_KUPO_D_ETAT_ENABLED" default:"false"`

	// AShantottoAscensionEnabled specifies whether the A Shantotto Ascension expansion is enabled
	AShantottoAscensionEnabled bool `env:"A_SHANTOTTO_ASCENSION_ENABLED" default:"false"`

	// VisionsOfAbysseaEnabled specifies whether the Visions of Abyssea expansion is enabled
	VisionsOfAbysseaEnabled bool `env:"VISIONS_OF_ABYSSEA_ENABLED" default:"false"`

	// ScarsOfAbysseaEnabled specifies whether the Scars of Abyssea expansion is enabled
	ScarsOfAbysseaEnabled bool `env:"SCARS_OF_ABYSSEA_ENABLED" default:"false"`

	// HeroesOfAbysseaEnabled specifies whether the Heroes of Abyssea expansion is enabled
	HeroesOfAbysseaEnabled bool `env:"HEROES_OF_ABYSSEA_ENABLED" default:"false"`

	// SeekersOfAdoulinEnabled specifies whether the Seekers of Adoulin expansion is enabled
	SeekersOfAdoulinEnabled bool `env:"SEEKERS_OF_ADOULIN_ENABLED" default:"false"`

	// SecureTokenEnabled specifies whether the secure token is enabled
	SecureTokenEnabled bool `env:"SECURE_TOKEN_ENABLED" default:"false"`

	// MogWardrobe3Enabled specifies whether the Mog Wardrobe 3 is enabled
	MogWardrobe3Enabled bool `env:"MOG_WARDROBE_3_ENABLED" default:"true"`

	// MogWardrobe4Enabled specifies whether the Mog Wardrobe 4 is enabled
	MogWardrobe4Enabled bool `env:"MOG_WARDROBE_4_ENABLED" default:"true"`

	// MogWardrobe5Enabled specifies whether the Mog Wardrobe 5 is enabled
	MogWardrobe5Enabled bool `env:"MOG_WARDROBE_5_ENABLED" default:"true"`

	// MogWardrobe6Enabled specifies whether the Mog Wardrobe 6 is enabled
	MogWardrobe6Enabled bool `env:"MOG_WARDROBE_6_ENABLED" default:"true"`

	// MogWardrobe7Enabled specifies whether the Mog Wardrobe 7 is enabled
	MogWardrobe7Enabled bool `env:"MOG_WARDROBE_7_ENABLED" default:"true"`

	// MogWardrobe8Enabled specifies whether the Mog Wardrobe 8 is enabled
	MogWardrobe8Enabled bool `env:"MOG_WARDROBE_8_ENABLED" default:"true"`

	// MapServerIP is the IP address of the map server to direct clients to
	MapServerIP string `env:"MAP_SERVER_IP" default:""`

	// MapServerPort is the port of the map server to direct clients to
	MapServerPort uint32 `env:"MAP_SERVER_PORT" default:"54230"`

	// SearchServerIP is the IP address of the search server to direct clients to
	SearchServerIP string `env:"SEARCH_SERVER_IP" default:""`

	// SearchServerPort is the port of the search server to direct clients to
	SearchServerPort uint32 `env:"SEARCH_SERVER_PORT" default:"54002"`
}

func ParseConfigFromEnv() Config {
	return env.Must(env.ParseAsWithOptions[Config](env.Options{
		DefaultValueTagName: "default",
	}))
}
