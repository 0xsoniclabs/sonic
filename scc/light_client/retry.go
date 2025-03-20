package light_client

import (
	"errors"
	"fmt"
	"time"

	"github.com/0xsoniclabs/sonic/scc"
	"github.com/0xsoniclabs/sonic/scc/cert"
	"github.com/Fantom-foundation/lachesis-base/inter/idx"
)

// retryProvider is used to wrap another provider and add a retry mechanism.
// It is a provider that retries provider methods, that returned an error, a
// specified number of times with some delay between retries up to a maximum timeout.
//
// Fields:
// - provider: The underlying provider to retry requests on.
// - retries: The maximum number of attempt to ask for certificates.
// - timeout: The time max time willing to wait for a request without error.
type retryProvider struct {
	provider provider
	retries  uint
	timeout  time.Duration
}

// newRetry creates a new retryProvider provider with the given provider and maximum
// number of retries and total timeout.
//
// Parameters:
//   - provider: The underlying provider to wrap with retry logic.
//   - retries: The maximum number of attempt to ask for certificates.
//   - timeout: The time max time willing to wait. if it is zero value,
//     then timeout is 10 seconds.
//
// Returns:
// - *retryProvider: A new retryProvider provider instance.
func newRetry(provider provider, retries uint, timeout time.Duration) *retryProvider {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &retryProvider{
		provider: provider,
		retries:  retries,
		timeout:  timeout,
	}
}

// Close closes the retryProvider.
// Closing an already closed provider has no effect.
func (r *retryProvider) close() {
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
func (r retryProvider) getCommitteeCertificates(first scc.Period, maxResults uint64) ([]cert.CommitteeCertificate, error) {
	return retry(r.retries, r.timeout, func() ([]cert.CommitteeCertificate, error) {
		return r.provider.getCommitteeCertificates(first, maxResults)
	})
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
func (r retryProvider) getBlockCertificates(first idx.Block, maxResults uint64) ([]cert.BlockCertificate, error) {
	return retry(r.retries, r.timeout, func() ([]cert.BlockCertificate, error) {
		return r.provider.getBlockCertificates(first, maxResults)
	})
}

// retry executes the given function a number of times equal to retries+1, unless
// one it returns a nil error, with incremental delays, waiting up to a max of timeout.
//
// Parameters:
// - fn: The function to execute and retry if failed.
//
// Returns:
//   - C: The result of the function if it succeeded.
//   - error: Nil if at least one execution of fn returned without error.
//     The joined error of all failed retries if all calls to fn failed.
func retry[C any](retries uint, timeout time.Duration, fn func() (C, error)) (C, error) {
	var errs []error
	now := time.Now()
	delay := time.Millisecond
	maxDelay := 128 * time.Millisecond
	for i := uint(0); i < retries+1; i++ {
		result, err := fn()
		if err == nil {
			return result, nil
		}
		errs = append(errs, err)
		if delay < maxDelay {
			delay *= 2
		}
		time.Sleep(delay)
		if time.Since(now) >= timeout {
			errs = append(errs, fmt.Errorf("exceeded timeout of %v", timeout))
			break
		}
	}

	var c C
	return c, errors.Join(fmt.Errorf("all retries failed: "), errors.Join(errs...))
}
