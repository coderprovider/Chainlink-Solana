package txm

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	solanaGo "github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/google/uuid"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/services"
	"github.com/smartcontractkit/chainlink-common/pkg/utils"

	"github.com/smartcontractkit/chainlink-solana/pkg/solana/client"
	"github.com/smartcontractkit/chainlink-solana/pkg/solana/config"
	"github.com/smartcontractkit/chainlink-solana/pkg/solana/fees"
)

const (
	MaxQueueLen      = 1000
	MaxRetryTimeMs   = 250 // max tx retry time (exponential retry will taper to retry every 0.25s)
	MaxSigsToConfirm = 256 // max number of signatures in GetSignatureStatus call
)

var _ services.Service = (*Txm)(nil)

//go:generate mockery --name SimpleKeystore --output ./mocks/ --case=underscore --filename simple_keystore.go
type SimpleKeystore interface {
	Sign(ctx context.Context, account string, data []byte) (signature []byte, err error)
}

// Txm manages transactions for the solana blockchain.
// simple implementation with no persistently stored txs
type Txm struct {
	starter services.StateMachine
	lggr    logger.Logger
	chSend  chan pendingTx
	chSim   chan pendingTx
	chStop  services.StopChan
	done    sync.WaitGroup
	cfg     config.Config
	txs     PendingTxContext
	ks      SimpleKeystore
	client  *utils.LazyLoad[client.ReaderWriter]
	fee     fees.Estimator
}

type pendingTx struct {
	tx        *solanaGo.Transaction
	timeout   time.Duration
	signature solanaGo.Signature
	id        uuid.UUID
}

// NewTxm creates a txm. Uses simulation so should only be used to send txes to trusted contracts i.e. OCR.
func NewTxm(chainID string, tc func() (client.ReaderWriter, error), cfg config.Config, ks SimpleKeystore, lggr logger.Logger) *Txm {
	return &Txm{
		lggr:   lggr,
		chSend: make(chan pendingTx, MaxQueueLen), // queue can support 1000 pending txs
		chSim:  make(chan pendingTx, MaxQueueLen), // queue can support 1000 pending txs
		chStop: make(chan struct{}),
		cfg:    cfg,
		txs:    newPendingTxContextWithProm(chainID),
		ks:     ks,
		client: utils.NewLazyLoad(tc),
	}
}

// Start subscribes to queuing channel and processes them.
func (txm *Txm) Start(ctx context.Context) error {
	return txm.starter.StartOnce("solana_txm", func() error {
		// determine estimator type
		var estimator fees.Estimator
		var err error
		switch strings.ToLower(txm.cfg.FeeEstimatorMode()) {
		case "fixed":
			estimator, err = fees.NewFixedPriceEstimator(txm.cfg)
		case "blockhistory":
			estimator, err = fees.NewBlockHistoryEstimator(txm.client, txm.cfg, txm.lggr)
		default:
			err = fmt.Errorf("unknown solana fee estimator type: %s", txm.cfg.FeeEstimatorMode())
		}
		if err != nil {
			return err
		}
		txm.fee = estimator
		if err := txm.fee.Start(ctx); err != nil {
			return err
		}

		txm.done.Add(3) // waitgroup: tx retry, confirmer, simulator
		go txm.run()
		return nil
	})
}

func (txm *Txm) run() {
	defer txm.done.Done()
	ctx, cancel := txm.chStop.NewCtx()
	defer cancel()

	// start confirmer + simulator
	go txm.confirm(ctx)
	go txm.simulate(ctx)

	for {
		select {
		case msg := <-txm.chSend:
			// process tx (pass tx copy)
			tx, id, sig, err := txm.sendWithRetry(ctx, *msg.tx, msg.timeout)
			if err != nil {
				txm.lggr.Errorw("failed to send transaction", "error", err)
				txm.client.Reset() // clear client if tx fails immediately (potentially bad RPC)
				continue           // skip remainining
			}

			// send tx + signature to simulation queue
			msg.tx = &tx
			msg.signature = sig
			msg.id = id
			select {
			case txm.chSim <- msg:
			default:
				txm.lggr.Warnw("failed to enqeue tx for simulation", "queueFull", len(txm.chSend) == MaxQueueLen, "tx", msg)
			}

			txm.lggr.Debugw("transaction sent", "signature", sig.String(), "id", id)
		case <-txm.chStop:
			return
		}
	}
}

func (txm *Txm) sendWithRetry(chanCtx context.Context, baseTx solanaGo.Transaction, timeout time.Duration) (solanaGo.Transaction, uuid.UUID, solanaGo.Signature, error) {
	// fetch client
	client, clientErr := txm.client.Get()
	if clientErr != nil {
		return solanaGo.Transaction{}, uuid.Nil, solanaGo.Signature{}, fmt.Errorf("failed to get client in soltxm.sendWithRetry: %w", clientErr)
	}

	// get key
	// fee payer account is index 0 account
	// https://github.com/gagliardetto/solana-go/blob/main/transaction.go#L252
	key := baseTx.Message.AccountKeys[0].String()

	// only calculate base price once
	// prevent underlying base changing when bumping (could occur with RPC based estimation)
	basePrice := txm.fee.BaseComputeUnitPrice()
	getFee := func(count uint) fees.ComputeUnitPrice {
		fee := fees.CalculateFee(
			basePrice,
			txm.cfg.ComputeUnitPriceMax(),
			txm.cfg.ComputeUnitPriceMin(),
			count,
		)
		return fees.ComputeUnitPrice(fee)
	}

	buildTx := func(base solanaGo.Transaction, retryCount uint) (solanaGo.Transaction, error) {
		newTx := base // make copy

		// set fee
		// fee bumping can be enabled by moving the setting & signing logic to the broadcaster
		if computeUnitErr := fees.SetComputeUnitPrice(&newTx, getFee(retryCount)); computeUnitErr != nil {
			return solanaGo.Transaction{}, computeUnitErr
		}

		// sign tx
		txMsg, marshalErr := newTx.Message.MarshalBinary()
		if marshalErr != nil {
			return solanaGo.Transaction{}, fmt.Errorf("error in soltxm.SendWithRetry.MarshalBinary: %w", marshalErr)
		}
		sigBytes, signErr := txm.ks.Sign(context.TODO(), key, txMsg)
		if signErr != nil {
			return solanaGo.Transaction{}, fmt.Errorf("error in soltxm.SendWithRetry.Sign: %w", signErr)
		}
		var finalSig [64]byte
		copy(finalSig[:], sigBytes)
		newTx.Signatures = append(newTx.Signatures, finalSig)

		return newTx, nil
	}

	initTx, initBuildErr := buildTx(baseTx, 0)
	if initBuildErr != nil {
		return solanaGo.Transaction{}, uuid.Nil, solanaGo.Signature{}, initBuildErr
	}

	// create timeout context
	ctx, cancel := context.WithTimeout(chanCtx, timeout)

	// send initial tx (do not retry and exit early if fails)
	sig, initSendErr := client.SendTx(ctx, &initTx)
	if initSendErr != nil {
		cancel()                           // cancel context when exiting early
		txm.txs.OnError(sig, TxFailReject) // increment failed metric
		return solanaGo.Transaction{}, uuid.Nil, solanaGo.Signature{}, fmt.Errorf("tx failed initial transmit: %w", initSendErr)
	}

	// store tx signature + cancel function
	id, initStoreErr := txm.txs.New(sig, cancel)
	if initStoreErr != nil {
		cancel() // cancel context when exiting early
		return solanaGo.Transaction{}, uuid.Nil, solanaGo.Signature{}, fmt.Errorf("failed to save tx signature (%s) to inflight txs: %w", sig, initStoreErr)
	}

	// used for tracking rebroadcasting only in SendWithRetry
	var sigs signatureList
	sigs.Allocate()
	if initSetErr := sigs.Set(0, sig); initSetErr != nil {
		return solanaGo.Transaction{}, uuid.Nil, solanaGo.Signature{}, fmt.Errorf("failed to save initial signature in signature list: %w", initSetErr)
	}

	txm.lggr.Debugw("tx initial broadcast", "id", id, "signature", sig)

	// retry with exponential backoff
	// until context cancelled by timeout or called externally
	// pass in copy of baseTx (used to build new tx with bumped fee) and broadcasted tx == initTx (used to retry tx without bumping)
	go func(baseTx, currentTx solanaGo.Transaction) {
		deltaT := 1 // ms
		tick := time.After(0)
		bumpCount := uint(0)
		bumpTime := time.Now()
		var wg sync.WaitGroup

		for {
			select {
			case <-ctx.Done():
				// stop sending tx after retry tx ctx times out (does not stop confirmation polling for tx)
				wg.Wait()
				txm.lggr.Debugw("stopped tx retry", "id", id, "signatures", sigs.List())
				return
			case <-tick:
				var shouldBump bool
				// bump if period > 0 and past time
				if txm.cfg.FeeBumpPeriod() != 0 && time.Since(bumpTime) > txm.cfg.FeeBumpPeriod() {
					bumpCount++
					bumpTime = time.Now()
					shouldBump = true
				}

				// if fee should be bumped, build new tx and replace currentTx
				if shouldBump {
					var retryBuildErr error
					currentTx, retryBuildErr = buildTx(baseTx, bumpCount)
					if retryBuildErr != nil {
						txm.lggr.Errorw("failed to build bumped retry tx", "error", retryBuildErr, "id", id)
						return // exit func if cannot build tx for retrying
					}
					ind := sigs.Allocate()
					if uint(ind) != bumpCount {
						txm.lggr.Errorw("INVARIANT VIOLATION: index (%d) != bumpCount (%d)", ind, bumpCount)
						return
					}
				}

				// take currentTx and broadcast, if bumped fee -> save signature to list
				wg.Add(1)
				go func(bump bool, count uint, retryTx solanaGo.Transaction) {
					defer wg.Done()

					retrySig, retrySendErr := client.SendTx(ctx, &retryTx)
					// this could occur if endpoint goes down or if ctx cancelled
					if retrySendErr != nil {
						if strings.Contains(retrySendErr.Error(), "context canceled") || strings.Contains(retrySendErr.Error(), "context deadline exceeded") {
							txm.lggr.Debugw("ctx error on send retry transaction", "error", retrySendErr, "signatures", sigs.List(), "id", id)
						} else {
							txm.lggr.Warnw("failed to send retry transaction", "error", retrySendErr, "signatures", sigs.List(), "id", id)
						}
						return
					}

					// save new signature if fee bumped
					if bump {
						if retryStoreErr := txm.txs.Add(id, retrySig); retryStoreErr != nil {
							txm.lggr.Warnw("error in adding retry transaction", "error", retryStoreErr, "id", id)
							return
						}
						if setErr := sigs.Set(int(count), retrySig); setErr != nil {
							// this should never happen
							txm.lggr.Errorw("INVARIANT VIOLATION", "error", setErr)
						}
						txm.lggr.Debugw("tx rebroadcast with bumped fee", "id", id, "fee", getFee(count), "signatures", sigs.List())
					}

					// prevent locking on waitgroup when ctx is closed
					wait := make(chan struct{})
					go func() {
						defer close(wait)
						sigs.Wait(int(count)) // wait until bump tx has set the tx signature to compare rebroadcast signatures
					}()
					select {
					case <-ctx.Done():
						return
					case <-wait:
					}

					// this should never happen (should match the signature saved to sigs)
					if fetchedSig, fetchErr := sigs.Get(int(count)); fetchErr != nil || retrySig != fetchedSig {
						txm.lggr.Errorw("original signature does not match retry signature", "expectedSignatures", sigs.List(), "receivedSignature", retrySig, "error", fetchErr)
					}
				}(shouldBump, bumpCount, currentTx)
			}

			// exponential increase in wait time, capped at 250ms
			deltaT *= 2
			if deltaT > MaxRetryTimeMs {
				deltaT = MaxRetryTimeMs
			}
			tick = time.After(time.Duration(deltaT) * time.Millisecond)
		}
	}(baseTx, initTx)

	// return signed tx, id, signature for use in simulation
	return initTx, id, sig, nil
}

// goroutine that polls to confirm implementation
// cancels the exponential retry once confirmed
func (txm *Txm) confirm(ctx context.Context) {
	defer txm.done.Done()

	tick := time.After(0)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick:
			// get list of tx signatures to confirm
			sigs := txm.txs.ListAll()

			// exit switch if not txs to confirm
			if len(sigs) == 0 {
				break
			}

			// get client
			client, err := txm.client.Get()
			if err != nil {
				txm.lggr.Errorw("failed to get client in soltxm.confirm", "error", err)
				break // exit switch
			}

			// batch sigs no more than MaxSigsToConfirm each
			sigsBatch, err := utils.BatchSplit(sigs, MaxSigsToConfirm)
			if err != nil { // this should never happen
				txm.lggr.Fatalw("failed to batch signatures", "error", err)
				break // exit switch
			}

			// process signatures
			processSigs := func(s []solanaGo.Signature, res []*rpc.SignatureStatusesResult) {
				// sort signatures and results process successful first
				s, res, err := SortSignaturesAndResults(s, res)
				if err != nil {
					txm.lggr.Errorw("sorting error", "error", err)
					return
				}

				for i := 0; i < len(res); i++ {
					// if status is nil (sig not found), continue polling
					// sig not found could mean invalid tx or not picked up yet
					if res[i] == nil {
						txm.lggr.Debugw("tx state: not found",
							"signature", s[i],
						)

						// check confirm timeout exceeded
						if txm.txs.Expired(s[i], txm.cfg.TxConfirmTimeout()) {
							id := txm.txs.OnError(s[i], TxFailDrop)
							txm.lggr.Infow("failed to find transaction within confirm timeout", "id", id, "signature", s[i], "timeoutSeconds", txm.cfg.TxConfirmTimeout())
						}
						continue
					}

					// if signature has an error, end polling
					if res[i].Err != nil {
						id := txm.txs.OnError(s[i], TxFailRevert)
						txm.lggr.Debugw("tx state: failed",
							"id", id,
							"signature", s[i],
							"error", res[i].Err,
							"status", res[i].ConfirmationStatus,
						)
						continue
					}

					// if signature is processed, keep polling
					if res[i].ConfirmationStatus == rpc.ConfirmationStatusProcessed {
						txm.lggr.Debugw("tx state: processed",
							"signature", s[i],
						)

						// check confirm timeout exceeded
						if txm.txs.Expired(s[i], txm.cfg.TxConfirmTimeout()) {
							id := txm.txs.OnError(s[i], TxFailDrop)
							txm.lggr.Debugw("tx failed to move beyond 'processed' within confirm timeout", "id", id, "signature", s[i], "timeoutSeconds", txm.cfg.TxConfirmTimeout())
						}
						continue
					}

					// if signature is confirmed/finalized, end polling
					if res[i].ConfirmationStatus == rpc.ConfirmationStatusConfirmed || res[i].ConfirmationStatus == rpc.ConfirmationStatusFinalized {
						id := txm.txs.OnSuccess(s[i])
						txm.lggr.Debugw(fmt.Sprintf("tx state: %s", res[i].ConfirmationStatus),
							"id", id,
							"signature", s[i],
						)
						continue
					}
				}
			}

			// waitgroup for processing
			var wg sync.WaitGroup
			wg.Add(len(sigsBatch))

			// loop through batch
			for i := 0; i < len(sigsBatch); i++ {
				// fetch signature statuses
				statuses, err := client.SignatureStatuses(ctx, sigsBatch[i])
				if err != nil {
					txm.lggr.Errorw("failed to get signature statuses in soltxm.confirm", "error", err)
					wg.Done() // don't block if exit early
					break     // exit for loop
				}

				// nonblocking: process batches as soon as they come in
				go func(index int) {
					defer wg.Done()
					processSigs(sigsBatch[index], statuses)
				}(i)
			}
			wg.Wait() // wait for processing to finish
		}
		tick = time.After(utils.WithJitter(txm.cfg.ConfirmPollPeriod()))
	}
}

// goroutine that simulates tx (use a bounded number of goroutines to pick from queue?)
// simulate can cancel the send retry function early in the tx management process
// additionally, it can provide reasons for why a tx failed in the logs
func (txm *Txm) simulate(ctx context.Context) {
	defer txm.done.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-txm.chSim:
			// get client
			client, err := txm.client.Get()
			if err != nil {
				txm.lggr.Errorw("failed to get client in soltxm.simulate", "error", err)
				continue
			}

			res, err := client.SimulateTx(ctx, msg.tx, nil) // use default options (does not verify signatures)
			if err != nil {
				// this error can occur if endpoint goes down or if invalid signature (invalid signature should occur further upstream in sendWithRetry)
				// allow retry to continue in case temporary endpoint failure (if still invalid, confirm or timeout will cleanup)
				txm.lggr.Debugw("failed to simulate tx", "id", msg.id, "signature", msg.signature, "error", err)
				continue
			}

			// continue if simulation does not return error continue
			if res.Err == nil {
				continue
			}

			// handle various errors
			// https://github.com/solana-labs/solana/blob/master/sdk/src/transaction/error.rs
			// ---
			errStr := fmt.Sprintf("%v", res.Err) // convert to string to handle various interfaces
			switch {
			// blockhash not found when simulating, occurs when network bank has not seen the given blockhash or tx is too old
			// let confirmation process clean up
			case strings.Contains(errStr, "BlockhashNotFound"):
				txm.lggr.Debugw("simulate: BlockhashNotFound", "id", msg.id, "signature", msg.signature, "result", res)
				continue
			// transaction will encounter execution error/revert, mark as reverted to remove from confirmation + retry
			case strings.Contains(errStr, "InstructionError"):
				txm.txs.OnError(msg.signature, TxFailSimRevert) // cancel retry
				txm.lggr.Debugw("simulate: InstructionError", "id", msg.id, "signature", msg.signature, "result", res)
				continue
			// transaction is already processed in the chain, letting txm confirmation handle
			case strings.Contains(errStr, "AlreadyProcessed"):
				txm.lggr.Debugw("simulate: AlreadyProcessed", "id", msg.id, "signature", msg.signature, "result", res)
				continue
			// unrecognized errors (indicates more concerning failures)
			default:
				txm.txs.OnError(msg.signature, TxFailSimOther) // cancel retry
				txm.lggr.Errorw("simulate: unrecognized error", "id", msg.id, "signature", msg.signature, "result", res)
				continue
			}
		}
	}
}

// Enqueue enqueue a msg destined for the solana chain.
func (txm *Txm) Enqueue(accountID string, tx *solanaGo.Transaction) error {
	// validate nil pointer
	if tx == nil {
		return errors.New("error in soltxm.Enqueue: tx is nil pointer")
	}
	// validate account keys slice
	if len(tx.Message.AccountKeys) == 0 {
		return errors.New("error in soltxm.Enqueue: not enough account keys in tx")
	}

	// validate expected key exists by trying to sign with it
	// fee payer account is index 0 account
	// https://github.com/gagliardetto/solana-go/blob/main/transaction.go#L252
	_, err := txm.ks.Sign(context.TODO(), tx.Message.AccountKeys[0].String(), nil)
	if err != nil {
		return fmt.Errorf("error in soltxm.Enqueue.GetKey: %w", err)
	}

	msg := pendingTx{
		tx:      tx,
		timeout: txm.cfg.TxRetryTimeout(),
	}

	select {
	case txm.chSend <- msg:
	default:
		txm.lggr.Errorw("failed to enqeue tx", "queueFull", len(txm.chSend) == MaxQueueLen, "tx", msg)
		return fmt.Errorf("failed to enqueue transaction for %s", accountID)
	}
	return nil
}

func (txm *Txm) InflightTxs() int {
	return len(txm.txs.ListAll())
}

// Close close service
func (txm *Txm) Close() error {
	return txm.starter.StopOnce("solanatxm", func() error {
		close(txm.chStop)
		txm.done.Wait()
		return txm.fee.Close()
	})
}
func (txm *Txm) Name() string { return "solanatxm" }

// Healthy service is healthy
func (txm *Txm) Healthy() error {
	return nil
}

// Ready service is ready
func (txm *Txm) Ready() error {
	return nil
}

func (txm *Txm) HealthReport() map[string]error { return map[string]error{txm.Name(): txm.Healthy()} }
