package exporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	commonMonitoring "github.com/smartcontractkit/chainlink-common/pkg/monitoring"
	"github.com/smartcontractkit/chainlink-common/pkg/utils/tests"

	"github.com/smartcontractkit/chainlink-solana/pkg/monitoring/metrics/mocks"
	"github.com/smartcontractkit/chainlink-solana/pkg/monitoring/testutils"
	"github.com/smartcontractkit/chainlink-solana/pkg/monitoring/types"
)

func TestReportObservations(t *testing.T) {
	ctx := tests.Context(t)
	lgr, logs := logger.TestObserved(t, zapcore.ErrorLevel)
	m := mocks.NewReportObservations(t)
	m.On("SetCount", mock.Anything, mock.Anything).Once()
	m.On("Cleanup", mock.Anything).Once()

	factory := NewReportObservationsFactory(lgr, m)

	chainConfig := testutils.GenerateChainConfig()
	feedConfig := testutils.GenerateFeedConfig()
	exporter, err := factory.NewExporter(commonMonitoring.ExporterParams{ChainConfig: chainConfig, FeedConfig: feedConfig, Nodes: []commonMonitoring.NodeConfig{}})
	require.NoError(t, err)

	// happy path
	exporter.Export(ctx, []types.TxDetails{{ObservationCount: 10}})
	exporter.Cleanup(ctx)

	// not txdetails type - no calls to mock
	assert.NotPanics(t, func() { exporter.Export(ctx, 1) })

	// zero txdetails - no calls to mock
	exporter.Export(ctx, []types.TxDetails{})

	// empty txdetails
	exporter.Export(ctx, []types.TxDetails{{}})
	assert.Equal(t, 1, logs.FilterMessage("exporter could not find non-empty TxDetails").Len())

	// multiple TxDetails should only call for the first non-empty one
	m.On("SetCount", uint8(1), mock.Anything).Once()
	exporter.Export(ctx, []types.TxDetails{{}, {ObservationCount: 1}, {ObservationCount: 10}})
}
