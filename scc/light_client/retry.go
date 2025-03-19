package light_client

import (
	"errors"
	"fmt"
	"time"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

// retryP is a provider that retries requests a certain number of times on failed
// requests for certificates with a determined wait between retries. It is used
// to wrap other providers to add a retry mechanism.
//
// Fields:
// - provider: The underlying provider to retry requests on.
// - attempts: The maximum number of attempt to ask for certificates.
// - delay: The time to wait between retries.
type retryP struct {
	provider provider
	attempts uint
	delay    time.Duration
}

// newRetry creates a new retryP provider with the given provider and maximum
// number of retries and delay between retries.
//
// Parameters:
// - provider: The underlying provider to wrap with retry logic.
// - attempts: The maximum number of attempt to ask for certificates.
// - delay: The time to wait between retries.
//
// Returns:
// - *retryP: A new retryP provider instance.
func newRetry(provider provider, attempts uint, delay time.Duration) *retryP {
	return &retryP{
		provider: provider,
		attempts: attempts,
		delay:    delay,
	}
}

// Close closes the retryP provider.
// Closing an already closed provider has no effect.
func (r *retryP) close() {
	r.provider.close()
}

// getCommitteeCertificates returns up to `maxResults` consecutive committee
// certificates starting from the given period.
//
// Parameters:
// - first: The starting period for which to retrieve committee certificates.
// - maxResults: The maximum number of committee certificates to retrieve.
//
// Returns:
// - []cert.CommitteeCertificate: A slice of committee certificates.
// - error: An error if the provider failed to obtain the requested certificates.
func (r retryP) getCommitteeCertificates(first scc.Period, maxResults uint64) ([]cert.CommitteeCertificate, error) {
	result, err := retry(r.attempts, r.delay, func() (any, error) {
		return r.provider.getCommitteeCertificates(first, maxResults)
	})
	if err != nil {
		return nil, err
	}
	return result.([]cert.CommitteeCertificate), nil
}

// getBlockCertificates returns up to `maxResults` consecutive block
// certificates starting from the given block number.
//
// Parameters:
// - first: The starting block number for which to retrieve the block certificate.
// - maxResults: The maximum number of block certificates to retrieve.
//
// Returns:
//   - []cert.BlockCertificate: The block certificates for the given block number
//     and the following blocks.
//   - error: An error if the provider failed to obtain the requested certificates.
func (r retryP) getBlockCertificates(first idx.Block, maxResults uint64) ([]cert.BlockCertificate, error) {
	result, err := retry(r.attempts, r.delay, func() (any, error) {
		return r.provider.getBlockCertificates(first, maxResults)
	})
	if err != nil {
		return nil, err
	}
	return result.([]cert.BlockCertificate), nil
}

// retry executes the given function for the given number of attempts, waiting
// the specified delay between attempts.
//
// Parameters:
// - fn: The function to execute and retry if failed.
//
// Returns:
//   - C: The result of the function if it succeeded.
//   - error: Nil if at least one execution of fn returned without error.
//     The joined error of all failed attempts if all attempts failed.
func retry[C any](attempts uint, delay time.Duration, fn func() (C, error)) (C, error) {
	var errs []error
	for i := uint(0); i < attempts; i++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		errs = append(errs, err)
		time.Sleep(delay)
	}

	var c C
	return c, errors.Join(fmt.Errorf("all retries failed: "), errors.Join(errs...))
}
