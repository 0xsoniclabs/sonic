package cert

// CommitteeCertificate is a certificate for a committee. This is an alias
// for cert.Certificate[cert.CommitteeStatement] to improve readability.
type CommitteeCertificate = Certificate[CommitteeStatement]

// CommitteeCertificate is a certificate for a block. This is an alias
// for cert.Certificate[cert.BlockStatement] to improve readability.
type BlockCertificate = Certificate[BlockStatement]
