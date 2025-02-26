package cert

import "github.com/0xsoniclabs/sonic/scc"

// Certificate is a message signed by a committee to certify the validity of a
// statement. A certificate is a claim produced by an authorized committee that
// a statement is true. In the certification chain, this is used to establish
// facts like the hash of a block at a certain height, the certificate chain's
// state at the begin of an epoch, or the composition of a committee for a
// certain period.
type Certificate[S Statement] struct {
	subject   S
	signature AggregatedSignature[S]
}

// NewCertificate creates a new certificate for the given statement. Initially,
// the certificate does not contain any signatures. Signatures can be added
// using the Add method.
func NewCertificate[S Statement](subject S) Certificate[S] {
	return Certificate[S]{subject: subject}
}

// NewCertificateWithSignature creates a new certificate for the given statement
// with the given aggregated signature. The aggregated signature shallow
// copied into the new certificate.
func NewCertificateWithSignature[S Statement](subject S,
	signature AggregatedSignature[S]) Certificate[S] {
	return Certificate[S]{subject: subject, signature: signature}
}

// Subject returns the statement that is certified by this certificate.
func (c *Certificate[S]) Subject() S {
	return c.subject
}

// Signature returns the aggregated signature of the certificate.
// The aggregated signature returned shallowed copied of the internal.
func (c *Certificate[S]) Signature() AggregatedSignature[S] {
	return c.signature
}

// Add adds a signature to the certificate for the given signer ID. The ID is
// used to identify signers in a certificate creation committee.
func (c *Certificate[S]) Add(id scc.MemberId, signature Signature[S]) error {
	return c.signature.Add(id, signature)
}

// Verify checks if the certificate is valid for the given committee. The
// certificate is valid if the committee is valid and the aggregated signature
// stored in this certificate has been signed by a 2/3 majority of the
// committee.
func (c *Certificate[S]) Verify(committee scc.Committee) error {
	return c.VerifyAuthority(committee, committee)
}

// VerifyAuthority checks if the certificate is valid for the given authority.
// The certificate is valid if the producer committee is valid and the
// aggregated signature stored in this certificate has been signed by a 2/3
// majority of the authority committee. This method is intended for fast-lane
// updates where committee updates of multiple periods can be skipped due to
// insignificant changes in the committee composition.
func (c *Certificate[S]) VerifyAuthority(authority, producers scc.Committee) error {
	return c.signature.Verify(authority, producers, c.subject)
}
