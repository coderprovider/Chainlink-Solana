package testconfig

import (
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/barkimedes/go-deepcopy"
	"github.com/google/uuid"
	"github.com/pelletier/go-toml/v2"
	"github.com/rs/zerolog"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/smartcontractkit/chainlink-common/pkg/config"
	"github.com/smartcontractkit/chainlink/integration-tests/types/config/node"
	"github.com/smartcontractkit/seth"

	ctf_config "github.com/smartcontractkit/chainlink-testing-framework/config"
	k8s_config "github.com/smartcontractkit/chainlink-testing-framework/k8s/config"
	"github.com/smartcontractkit/chainlink-testing-framework/logging"
	"github.com/smartcontractkit/chainlink-testing-framework/utils/osutil"
	"github.com/smartcontractkit/chainlink-testing-framework/utils/ptr"

	ocr2_config "github.com/smartcontractkit/chainlink-solana/integration-tests/testconfig/ocr2"
	solcfg "github.com/smartcontractkit/chainlink-solana/pkg/solana/config"
)

type TestConfig struct {
	ChainlinkImage        *ctf_config.ChainlinkImageConfig `toml:"ChainlinkImage"`
	Logging               *ctf_config.LoggingConfig        `toml:"Logging"`
	ChainlinkUpgradeImage *ctf_config.ChainlinkImageConfig `toml:"ChainlinkUpgradeImage"`
	Network               *ctf_config.NetworkConfig        `toml:"Network"`
	Common                *Common                          `toml:"Common"`
	OCR2                  *ocr2_config.Config              `toml:"OCR2"`
	SolanaConfig          *SolanaConfig                    `toml:"SolanaConfig"`
	ConfigurationName     string                           `toml:"-"`

	// getter funcs for passing parameters
	GetChainID func() string
	GetURL     func() string
}

func (c *TestConfig) GetLoggingConfig() *ctf_config.LoggingConfig {
	return c.Logging
}

func (c *TestConfig) GetPrivateEthereumNetworkConfig() *ctf_config.EthereumNetworkConfig {
	return &ctf_config.EthereumNetworkConfig{}
}

func (c *TestConfig) GetPyroscopeConfig() *ctf_config.PyroscopeConfig {
	return &ctf_config.PyroscopeConfig{}
}

func (c *TestConfig) GetSethConfig() *seth.Config {
	return nil
}

func (c *TestConfig) GetNodeConfig() *ctf_config.NodeConfig {
	cfgTOML, err := c.GetNodeConfigTOML()
	if err != nil {
		log.Fatalf("failed to parse TOML config: %s", err)
		return nil
	}

	return &ctf_config.NodeConfig{
		BaseConfigTOML: cfgTOML,
	}
}

func (c *TestConfig) GetNodeConfigTOML() (string, error) {
	var chainID, url string
	if c.GetChainID != nil {
		chainID = c.GetChainID()
	}
	if c.GetURL != nil {
		url = c.GetURL()
	}

	solConfig := solcfg.TOMLConfig{
		Enabled: ptr.Ptr(true),
		ChainID: ptr.Ptr(chainID),
		Nodes: []*solcfg.Node{
			{
				Name: ptr.Ptr("primary"),
				URL:  config.MustParseURL(url),
			},
		},
	}
	baseConfig := node.NewBaseConfig()
	baseConfig.Solana = solcfg.TOMLConfigs{
		&solConfig,
	}
	baseConfig.OCR2.Enabled = ptr.Ptr(true)
	baseConfig.P2P.V2.Enabled = ptr.Ptr(true)
	fiveSecondDuration := config.MustNewDuration(5 * time.Second)

	baseConfig.P2P.V2.DeltaDial = fiveSecondDuration
	baseConfig.P2P.V2.DeltaReconcile = fiveSecondDuration
	baseConfig.P2P.V2.ListenAddresses = &[]string{"0.0.0.0:6690"}

	return baseConfig.TOMLString()
}

var embeddedConfigs embed.FS
var areConfigsEmbedded bool

func init() {
	embeddedConfigs = embeddedConfigsFs
}

// Saves Test Config to a local file
func (c *TestConfig) Save() (string, error) {
	filePath := fmt.Sprintf("test_config-%s.toml", uuid.New())

	content, err := toml.Marshal(*c)
	if err != nil {
		return "", fmt.Errorf("error marshaling test config: %w", err)
	}

	err = os.WriteFile(filePath, content, 0600)
	if err != nil {
		return "", fmt.Errorf("error writing test config: %w", err)
	}

	return filePath, nil
}

// MustCopy Returns a deep copy of the Test Config or panics on error
func (c TestConfig) MustCopy() any {
	return deepcopy.MustAnything(c).(TestConfig)
}

// MustCopy Returns a deep copy of struct passed to it and returns a typed copy (or panics on error)
func MustCopy[T any](c T) T {
	return deepcopy.MustAnything(c).(T)
}

func (c TestConfig) GetNetworkConfig() *ctf_config.NetworkConfig {
	return c.Network
}

func (c TestConfig) GetChainlinkImageConfig() *ctf_config.ChainlinkImageConfig {
	return c.ChainlinkImage
}

func (c TestConfig) GetCommonConfig() *Common {
	return c.Common
}

func (c TestConfig) GetChainlinkUpgradeImageConfig() *ctf_config.ChainlinkImageConfig {
	return c.ChainlinkUpgradeImage
}

func (c TestConfig) GetConfigurationName() string {
	return c.ConfigurationName
}

func (c *TestConfig) AsBase64() (string, error) {
	content, err := toml.Marshal(*c)
	if err != nil {
		return "", fmt.Errorf("error marshaling test config: %w", err)
	}

	return base64.StdEncoding.EncodeToString(content), nil
}

type Common struct {
	Network   *string `toml:"network"`
	InsideK8s *bool   `toml:"inside_k8"`
	User      *string `toml:"user"`
	// if rpc requires api key to be passed as an HTTP header
	RPCURL             *string `toml:"rpc_url"`
	WsURL              *string `toml:"ws_url"`
	PrivateKey         *string `toml:"private_key"`
	Stateful           *bool   `toml:"stateful_db"`
	InternalDockerRepo *string `toml:"internal_docker_repo"`
	DevnetImage        *string `toml:"devnet_image"`
}

type SolanaConfig struct {
	Secret                    *string `toml:"secret"`
	OCR2ProgramID             *string `toml:"ocr2_program_id"`
	AccessControllerProgramID *string `toml:"access_controller_program_id"`
	StoreProgramID            *string `toml:"store_program_id"`
	LinkTokenAddress          *string `toml:"link_token_address"`
	VaultAddress              *string `toml:"vault_address"`
}

func (c *SolanaConfig) Validate() error {
	if c.Secret == nil {
		return fmt.Errorf("secret must be set")
	}
	if c.OCR2ProgramID == nil {
		return fmt.Errorf("ocr2_program_id must be set")
	}
	if c.AccessControllerProgramID == nil {
		return fmt.Errorf("access_controller_program_id must be set")
	}
	if c.StoreProgramID == nil {
		return fmt.Errorf("store_program_id must be set")
	}
	if c.LinkTokenAddress == nil {
		return fmt.Errorf("link_token_address must be set")
	}
	if c.VaultAddress == nil {
		return fmt.Errorf("vault_address must be set")
	}
	return nil
}

func (c *Common) Validate() error {
	if c.Network == nil {
		return fmt.Errorf("network must be set")
	}

	switch *c.Network {
	case "localnet":
		if c.DevnetImage == nil {
			return fmt.Errorf("devnet_image must be set")
		}
	case "devnet":
		if c.PrivateKey == nil {
			return fmt.Errorf("private_key must be set")
		}
		if c.RPCURL == nil {
			return fmt.Errorf("rpc_url must be set")
		}
		if c.WsURL == nil {
			return fmt.Errorf("rpc_url must be set")
		}

	default:
		return fmt.Errorf("network must be either 'localnet' or 'devnet'")
	}

	if c.InsideK8s == nil {
		return fmt.Errorf("inside_k8 must be set")
	}

	if c.InternalDockerRepo == nil {
		return fmt.Errorf("internal_docker_repo must be set")
	}

	err := os.Setenv("INTERNAL_DOCKER_REPO", *c.InternalDockerRepo)
	if err != nil {
		return fmt.Errorf("could not set INTERNAL_DOCKER_REPO env var")
	}

	if c.User == nil {
		return fmt.Errorf("user must be set")
	}

	err = os.Setenv("CHAINLINK_ENV_USER", *c.User)
	if err != nil {
		return fmt.Errorf("could not set CHAINLINK_ENV_USER env var")
	}

	if c.Stateful == nil {
		return fmt.Errorf("stateful_db state for db must be set")
	}

	return nil
}

type Product string

const (
	OCR2 Product = "ocr2"
)

const TestTypeEnvVarName = "TEST_TYPE"

const (
	Base64OverrideEnvVarName = k8s_config.EnvBase64ConfigOverride
	NoKey                    = "NO_KEY"
)

func GetConfig(configurationName string, product Product) (TestConfig, error) {
	logger := logging.GetTestLogger(nil)

	configurationName = strings.ReplaceAll(configurationName, "/", "_")
	configurationName = strings.ReplaceAll(configurationName, " ", "_")
	configurationName = cases.Title(language.English, cases.NoLower).String(configurationName)
	fileNames := []string{
		"default.toml",
		fmt.Sprintf("%s.toml", product),
		"overrides.toml",
	}

	testConfig := TestConfig{}
	testConfig.ConfigurationName = configurationName
	logger.Debug().Msgf("Will apply configuration named '%s' if it is found in any of the configs", configurationName)

	var handleSpecialOverrides = func(logger zerolog.Logger, filename, configurationName string, target *TestConfig, content []byte, product Product) error {
		switch product {
		default:
			err := ctf_config.BytesToAnyTomlStruct(logger, filename, configurationName, target, content)
			if err != nil {
				return fmt.Errorf("error reading file %s: %w", filename, err)
			}

			return nil
		}
	}

	// read embedded configs is build tag "embed" is set
	// this makes our life much easier when using a binary
	if areConfigsEmbedded {
		logger.Info().Msg("Reading embedded configs")
		embeddedFiles := []string{"default.toml", fmt.Sprintf("%s/%s.toml", product, product)}
		for _, fileName := range embeddedFiles {
			file, err := embeddedConfigs.ReadFile(fileName)
			if err != nil && errors.Is(err, os.ErrNotExist) {
				logger.Debug().Msgf("Embedded config file %s not found. Continuing", fileName)
				continue
			} else if err != nil {
				return TestConfig{}, fmt.Errorf("error reading embedded config: %w", err)
			}

			err = handleSpecialOverrides(logger, fileName, "", &testConfig, file, product) // use empty configurationName to read default config
			if err != nil {
				return TestConfig{}, fmt.Errorf("error unmarshalling embedded config: %w", err)
			}
		}
	}

	logger.Info().Msg("Reading configs from file system")
	for _, fileName := range fileNames {
		logger.Debug().Msgf("Looking for config file %s", fileName)
		filePath, err := osutil.FindFile(fileName, osutil.DEFAULT_STOP_FILE_NAME, 3)

		if err != nil && errors.Is(err, os.ErrNotExist) {
			logger.Debug().Msgf("Config file %s not found", fileName)
			continue
		} else if err != nil {
			return TestConfig{}, fmt.Errorf("error looking for file %s: %w", filePath, err)
		}
		logger.Debug().Str("location", filePath).Msgf("Found config file %s", fileName)

		content, err := readFile(filePath)
		if err != nil {
			return TestConfig{}, fmt.Errorf("error reading file %s: %w", filePath, err)
		}

		err = handleSpecialOverrides(logger, fileName, "", &testConfig, content, product) // use empty configurationName to read default config
		if err != nil {
			return TestConfig{}, fmt.Errorf("error reading file %s: %w", filePath, err)
		}
	}

	logger.Info().Msg("Reading configs from Base64 override env var")
	configEncoded, isSet := os.LookupEnv(Base64OverrideEnvVarName)
	if isSet && configEncoded != "" {
		logger.Debug().Msgf("Found base64 config override environment variable '%s' found", Base64OverrideEnvVarName)
		decoded, err := base64.StdEncoding.DecodeString(configEncoded)
		if err != nil {
			return TestConfig{}, err
		}

		err = handleSpecialOverrides(logger, Base64OverrideEnvVarName, "", &testConfig, decoded, product) // use empty configurationName to read default config
		if err != nil {
			return TestConfig{}, fmt.Errorf("error unmarshaling base64 config: %w", err)
		}
	} else {
		logger.Debug().Msg("Base64 config override from environment variable not found")
	}

	// it neede some custom logic, so we do it separately
	err := testConfig.readNetworkConfiguration()
	if err != nil {
		return TestConfig{}, fmt.Errorf("error reading network config: %w", err)
	}

	logger.Debug().Msg("Validating test config")
	err = testConfig.Validate()
	if err != nil {
		return TestConfig{}, fmt.Errorf("error validating test config: %w", err)
	}

	if testConfig.Common == nil {
		testConfig.Common = &Common{}
	}

	logger.Debug().Msg("Correct test config constructed successfully")
	return testConfig, nil
}

func (c *TestConfig) readNetworkConfiguration() error {
	// currently we need to read that kind of secrets only for network configuration
	if c == nil {
		c.Network = &ctf_config.NetworkConfig{}
	}

	c.Network.UpperCaseNetworkNames()
	err := c.Network.Default()
	if err != nil {
		return fmt.Errorf("error reading default network config: %w", err)
	}

	return nil
}

func (c *TestConfig) Validate() error {
	defer func() {
		if r := recover(); r != nil {
			panic(fmt.Errorf("Panic during test config validation: '%v'. Most probably due to presence of partial product config", r))
		}
	}()
	if c.ChainlinkImage == nil {
		return fmt.Errorf("chainlink image config must be set")
	}
	if err := c.ChainlinkImage.Validate(); err != nil {
		return fmt.Errorf("chainlink image config validation failed: %w", err)
	}
	if c.ChainlinkUpgradeImage != nil {
		if err := c.ChainlinkUpgradeImage.Validate(); err != nil {
			return fmt.Errorf("chainlink upgrade image config validation failed: %w", err)
		}
	}
	if err := c.Network.Validate(); err != nil {
		return fmt.Errorf("network config validation failed: %w", err)
	}

	if c.Common == nil {
		return fmt.Errorf("common config must be set")
	}

	if err := c.Common.Validate(); err != nil {
		return fmt.Errorf("Common config validation failed: %w", err)
	}

	if c.OCR2 == nil {
		return fmt.Errorf("OCR2 config must be set")
	}

	if err := c.OCR2.Validate(); err != nil {
		return fmt.Errorf("OCR2 config validation failed: %w", err)
	}
	if c.SolanaConfig == nil {
		return fmt.Errorf("SolanaConfig config must be set")
	}

	if err := c.SolanaConfig.Validate(); err != nil {
		return fmt.Errorf("SolanaConfig config validation failed: %w", err)
	}
	return nil
}

func readFile(filePath string) ([]byte, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	return content, nil
}
