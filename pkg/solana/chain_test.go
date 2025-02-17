package solana

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-common/pkg/config"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-solana/pkg/solana/client"
	solcfg "github.com/smartcontractkit/chainlink-solana/pkg/solana/config"
)

const TestSolanaGenesisHashTemplate = `{"jsonrpc":"2.0","result":"%s","id":1}`

func TestSolanaChain_GetClient(t *testing.T) {
	checkOnce := map[string]struct{}{}
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out := fmt.Sprintf(TestSolanaGenesisHashTemplate, client.MainnetGenesisHash) // mainnet genesis hash

		if !strings.Contains(r.URL.Path, "/mismatch") {
			// devnet gensis hash
			out = fmt.Sprintf(TestSolanaGenesisHashTemplate, client.DevnetGenesisHash)

			// clients with correct chainID should request chainID only once
			if _, exists := checkOnce[r.URL.Path]; exists {
				assert.NoError(t, fmt.Errorf("rpc has been called once already for successful client '%s'", r.URL.Path))
			}
			checkOnce[r.URL.Path] = struct{}{}
		}

		_, err := w.Write([]byte(out))
		require.NoError(t, err)
	}))
	defer mockServer.Close()

	ch := solcfg.Chain{}
	ch.SetDefaults()
	cfg := &solcfg.TOMLConfig{
		ChainID: ptr("devnet"),
		Chain:   ch,
	}
	testChain := chain{
		id:          "devnet",
		cfg:         cfg,
		lggr:        logger.Test(t),
		clientCache: map[string]*verifiedCachedClient{},
	}

	cfg.Nodes = []*solcfg.Node{
		{
			Name: ptr("devnet"),
			URL:  config.MustParseURL(mockServer.URL + "/1"),
		},
		{
			Name: ptr("devnet"),
			URL:  config.MustParseURL(mockServer.URL + "/2"),
		},
	}
	_, err := testChain.getClient()
	assert.NoError(t, err)

	// random nodes (happy path, 1 valid + multiple invalid)
	cfg.Nodes = []*solcfg.Node{
		{
			Name: ptr("devnet"),
			URL:  config.MustParseURL(mockServer.URL + "/1"),
		},
		{
			Name: ptr("devnet"),
			URL:  config.MustParseURL(mockServer.URL + "/mismatch/1"),
		},
		{
			Name: ptr("devnet"),
			URL:  config.MustParseURL(mockServer.URL + "/mismatch/2"),
		},
		{
			Name: ptr("devnet"),
			URL:  config.MustParseURL(mockServer.URL + "/mismatch/3"),
		},
		{
			Name: ptr("devnet"),
			URL:  config.MustParseURL(mockServer.URL + "/mismatch/4"),
		},
	}
	_, err = testChain.getClient()
	assert.NoError(t, err)

	// empty nodes response
	cfg.Nodes = nil
	_, err = testChain.getClient()
	assert.Error(t, err)

	// no valid nodes to select from
	cfg.Nodes = []*solcfg.Node{
		{
			Name: ptr("devnet"),
			URL:  config.MustParseURL(mockServer.URL + "/mismatch/1"),
		},
		{
			Name: ptr("devnet"),
			URL:  config.MustParseURL(mockServer.URL + "/mismatch/2"),
		},
	}
	_, err = testChain.getClient()
	assert.NoError(t, err)
}

func TestSolanaChain_VerifiedClient(t *testing.T) {
	called := false
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out := `{ "jsonrpc": "2.0", "result": 1234, "id": 1 }` // getSlot response

		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)

		// handle getGenesisHash request
		if strings.Contains(string(body), "getGenesisHash") {
			// should only be called once, chainID will be cached in chain
			// allowing `mismatch` to be ignored, since invalid nodes will try to verify the chain ID
			// if it is not verified
			if !strings.Contains(r.URL.Path, "/mismatch") && called {
				assert.NoError(t, errors.New("rpc has been called once already"))
			}
			// devnet genesis hash
			out = fmt.Sprintf(TestSolanaGenesisHashTemplate, client.DevnetGenesisHash)
		}
		_, err = w.Write([]byte(out))
		require.NoError(t, err)
		called = true
	}))
	defer mockServer.Close()

	ch := solcfg.Chain{}
	ch.SetDefaults()
	cfg := &solcfg.TOMLConfig{
		ChainID: ptr("devnet"),
		Chain:   ch,
	}
	testChain := chain{
		cfg:         cfg,
		lggr:        logger.Test(t),
		clientCache: map[string]*verifiedCachedClient{},
	}
	nName := t.Name() + "-" + uuid.NewString()
	node := &solcfg.Node{
		Name: &nName,
		URL:  config.MustParseURL(mockServer.URL),
	}

	// happy path
	testChain.id = "devnet"
	_, err := testChain.verifiedClient(node)
	require.NoError(t, err)

	// retrieve cached client and retrieve slot height
	c, err := testChain.verifiedClient(node)
	require.NoError(t, err)
	slot, err := c.SlotHeight()
	assert.NoError(t, err)
	assert.Equal(t, uint64(1234), slot)

	node.URL = config.MustParseURL(mockServer.URL + "/mismatch")
	testChain.id = "incorrect"
	c, err = testChain.verifiedClient(node)
	assert.NoError(t, err)
	_, err = c.ChainID()
	// expect error from id mismatch (even if using a cached client) when performing RPC calls
	assert.Error(t, err)
	assert.Equal(t, fmt.Sprintf("client returned mismatched chain id (expected: %s, got: %s): %s", "incorrect", "devnet", node.URL), err.Error())
}

func TestSolanaChain_VerifiedClient_ParallelClients(t *testing.T) {
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		out := fmt.Sprintf(TestSolanaGenesisHashTemplate, client.DevnetGenesisHash)
		_, err := w.Write([]byte(out))
		require.NoError(t, err)
	}))
	defer mockServer.Close()

	ch := solcfg.Chain{}
	ch.SetDefaults()
	cfg := &solcfg.TOMLConfig{
		ChainID: ptr("devnet"),
		Enabled: ptr(true),
		Chain:   ch,
	}
	testChain := chain{
		id:          "devnet",
		cfg:         cfg,
		lggr:        logger.Test(t),
		clientCache: map[string]*verifiedCachedClient{},
	}
	nName := t.Name() + "-" + uuid.NewString()
	node := &solcfg.Node{
		Name: &nName,
		URL:  config.MustParseURL(mockServer.URL),
	}

	var wg sync.WaitGroup
	wg.Add(2)

	var client0 client.ReaderWriter
	var client1 client.ReaderWriter
	var err0 error
	var err1 error

	// call verifiedClient in parallel
	go func() {
		client0, err0 = testChain.verifiedClient(node)
		assert.NoError(t, err0)
		wg.Done()
	}()
	go func() {
		client1, err1 = testChain.verifiedClient(node)
		assert.NoError(t, err1)
		wg.Done()
	}()

	wg.Wait()

	// check if pointers are all the same
	assert.Equal(t, testChain.clientCache[mockServer.URL], client0)
	assert.Equal(t, testChain.clientCache[mockServer.URL], client1)
}

func ptr[T any](t T) *T {
	return &t
}
