package monitoring

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/gagliardetto/solana-go"
)

func SolanaFeedParser(buf io.ReadCloser) ([]FeedConfig, error) {
	rawFeeds := []SolanaFeedConfig{}
	decoder := json.NewDecoder(buf)
	if err := decoder.Decode(&rawFeeds); err != nil {
		return nil, fmt.Errorf("unable to unmarshal feeds config data: %w", err)
	}
	feeds := make([]FeedConfig, len(rawFeeds))
	for i, rawFeed := range rawFeeds {
		contractAddress, err := solana.PublicKeyFromBase58(rawFeed.ContractAddressBase58)
		if err != nil {
			return nil, fmt.Errorf("failed to parse program id '%s' from JSON at index i=%d: %w", rawFeed.ContractAddressBase58, i, err)
		}
		transmissionsAccount, err := solana.PublicKeyFromBase58(rawFeed.TransmissionsAccountBase58)
		if err != nil {
			return nil, fmt.Errorf("failed to parse transmission account '%s' from JSON at index i=%d: %w", rawFeed.TransmissionsAccountBase58, i, err)
		}
		stateAccount, err := solana.PublicKeyFromBase58(rawFeed.StateAccountBase58)
		if err != nil {
			return nil, fmt.Errorf("failed to parse state account '%s' from JSON at index i=%d: %w", rawFeed.StateAccountBase58, i, err)
		}
		feeds[i] = FeedConfig(SolanaFeedConfig{
			rawFeed.Name,
			rawFeed.Path,
			rawFeed.Symbol,
			rawFeed.HeartbeatSec,
			rawFeed.ContractType,
			rawFeed.ContractStatus,

			rawFeed.ContractAddressBase58,
			rawFeed.TransmissionsAccountBase58,
			rawFeed.StateAccountBase58,

			contractAddress,
			transmissionsAccount,
			stateAccount,
		})
	}
	return feeds, nil
}

type SolanaFeedConfig struct {
	Name           string `json:"name,omitempty"`
	Path           string `json:"path,omitempty"`
	Symbol         string `json:"symbol,omitempty"`
	HeartbeatSec   int64  `json:"heartbeat,omitempty"`
	ContractType   string `json:"contract_type,omitempty"`
	ContractStatus string `json:"status,omitempty"`

	ContractAddressBase58      string `json:"contract_address_base58,omitempty"`
	TransmissionsAccountBase58 string `json:"transmissions_account_base58,omitempty"`
	StateAccountBase58         string `json:"state_account_base58,omitempty"`

	ContractAddress      solana.PublicKey `json:"-"`
	TransmissionsAccount solana.PublicKey `json:"-"`
	StateAccount         solana.PublicKey `json:"-"`
}

var _ FeedConfig = SolanaFeedConfig{}

func (s SolanaFeedConfig) GetName() string {
	return s.Name
}

func (s SolanaFeedConfig) GetPath() string {
	return s.Path
}

func (s SolanaFeedConfig) GetFeedPath() string {
	return s.Path
}

func (s SolanaFeedConfig) GetSymbol() string {
	return s.Symbol
}

func (s SolanaFeedConfig) GetHeartbeatSec() int64 {
	return s.HeartbeatSec
}

func (s SolanaFeedConfig) GetContractType() string {
	return s.ContractType
}

func (s SolanaFeedConfig) GetContractStatus() string {
	return s.ContractStatus
}

func (s SolanaFeedConfig) GetContractAddress() string {
	return s.ContractAddress.String()
}

func (s SolanaFeedConfig) GetContractAddressBytes() []byte {
	return s.ContractAddress.Bytes()
}

func (s SolanaFeedConfig) ToMapping() map[string]interface{} {
	return map[string]interface{}{
		"feed_name":             s.Name,
		"feed_path":             s.Path,
		"symbol":                s.Symbol,
		"heartbeat_sec":         int64(s.HeartbeatSec),
		"contract_type":         s.ContractType,
		"contract_status":       s.ContractStatus,
		"contract_address":      s.ContractAddress.Bytes(),
		"transmissions_account": s.TransmissionsAccount.Bytes(),
		"state_account":         s.StateAccount.Bytes(),
	}
}