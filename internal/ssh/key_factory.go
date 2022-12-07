package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"

	"golang.org/x/crypto/ssh"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/logging"
)

const (
	rsaPrivateKeyType = "RSA PRIVATE KEY"
)

// ErrInvalidPrivateKey will be used to wrap a RSA internal error
type ErrInvalidPrivateKey struct {
	inner error
}

func (e *ErrInvalidPrivateKey) Error() string {
	return fmt.Sprintf("invalid private key: %v", e.inner)
}

func (e *ErrInvalidPrivateKey) Unwrap() error {
	return e.inner
}

func (e *ErrInvalidPrivateKey) Is(err error) bool {
	_, ok := err.(*ErrInvalidPrivateKey)
	return ok
}

// KeyFactory is a factory for Public and Private key pairs
type KeyFactory interface {
	Create(bitSize int) (*KeyPair, error)
}

// KeyPair centralizes the generated public and private keys
type KeyPair struct {
	PublicKey  []byte
	PrivateKey []byte
}

type keyFactory struct {
	logger logging.Logger

	// Functions encapsulated to make easier creating unit tests
	generateRSAKey  func(random io.Reader, bits int) (*rsa.PrivateKey, error)
	newSSHPublicKey func(key interface{}) (ssh.PublicKey, error)
}

// NewKeyFactory instantiates a concrete instance of KeyFactory
func NewKeyFactory(logger logging.Logger) KeyFactory {
	return &keyFactory{
		logger:          logger,
		generateRSAKey:  rsa.GenerateKey,
		newSSHPublicKey: ssh.NewPublicKey,
	}
}

// Create generates a RSA Public and Private key pair
func (r *keyFactory) Create(bitSize int) (*KeyPair, error) {
	r.logger.Debug("[Create] Will generate new key pair")

	privateKey, err := r.generateRSAKey(rand.Reader, bitSize)
	if err != nil {
		return nil, fmt.Errorf("generating the private key: %w", err)
	}

	err = privateKey.Validate()
	if err != nil {
		return nil, &ErrInvalidPrivateKey{inner: err}
	}

	privateKeyBytes := r.encodePrivateKeyToPEM(privateKey)

	publicKeyBytes, err := r.getPublicKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("generating the public key: %w", err)
	}

	keyPair := &KeyPair{
		PublicKey:  publicKeyBytes,
		PrivateKey: privateKeyBytes,
	}

	r.logger.Debug("[Create] Key pair generated with success")

	return keyPair, nil
}

func (r *keyFactory) encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	block := pem.Block{
		Type:    rsaPrivateKeyType,
		Headers: nil,
		Bytes:   x509.MarshalPKCS1PrivateKey(privateKey),
	}

	return pem.EncodeToMemory(&block)
}

func (r *keyFactory) getPublicKey(privateKey *rsa.PrivateKey) ([]byte, error) {
	publicKey, err := r.newSSHPublicKey(&privateKey.PublicKey)
	if err != nil {
		return nil, err
	}

	return ssh.MarshalAuthorizedKey(publicKey), nil
}
