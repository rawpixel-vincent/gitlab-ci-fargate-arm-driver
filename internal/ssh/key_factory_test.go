package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ssh"

	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/assertions"
	"gitlab.com/gitlab-org/ci-cd/custom-executor-drivers/fargate/internal/logging/test"
)

func TestNewKeyFactory(t *testing.T) {
	testLogger := test.NewNullLogger()

	factory := NewKeyFactory(testLogger)

	assert.NotNil(t, factory)
	assert.Equal(t, testLogger, factory.(*keyFactory).logger)
	assert.NotNil(t, factory.(*keyFactory).generateRSAKey)
	assert.NotNil(t, factory.(*keyFactory).newSSHPublicKey)
}

func TestKeyFactory_Create(t *testing.T) {
	testError := errors.New("simulated error")
	testErrorRSAInternal := new(ErrInvalidPrivateKey)
	bitSize := 64

	validKeyPair, err := rsa.GenerateKey(rand.Reader, bitSize)
	require.NoError(t, err)

	invalidKeyPair := new(rsa.PrivateKey)

	tests := map[string]struct {
		mockedRSAKey   *rsa.PrivateKey
		rsaKeyError    error
		publicKeyError error
		expectedError  error
	}{
		"Key pair generated with success": {
			mockedRSAKey:   validKeyPair,
			rsaKeyError:    nil,
			publicKeyError: nil,
			expectedError:  nil,
		},
		"Error generating key": {
			mockedRSAKey:   nil,
			rsaKeyError:    testError,
			publicKeyError: nil,
			expectedError:  testError,
		},
		"Error invalid key pair": {
			mockedRSAKey:   invalidKeyPair,
			rsaKeyError:    nil,
			publicKeyError: nil,
			expectedError:  testErrorRSAInternal,
		},
		"Error creating SSH public key": {
			mockedRSAKey:   validKeyPair,
			rsaKeyError:    nil,
			publicKeyError: testError,
			expectedError:  testError,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			factory := NewKeyFactory(test.NewNullLogger())

			factory.(*keyFactory).generateRSAKey = func(r io.Reader, b int) (*rsa.PrivateKey, error) {
				return tt.mockedRSAKey, tt.rsaKeyError
			}

			if tt.publicKeyError != nil {
				factory.(*keyFactory).newSSHPublicKey = func(k interface{}) (ssh.PublicKey, error) {
					return nil, tt.publicKeyError
				}
			}

			keyPair, err := factory.Create(bitSize)

			if tt.expectedError != nil {
				assertions.ErrorIs(t, err, tt.expectedError)
				assert.Nil(t, keyPair)
				return
			}

			assert.NoError(t, err)
			require.NotNil(t, keyPair)
			assert.NotEmpty(t, keyPair.PublicKey)
			assert.NotEmpty(t, keyPair.PrivateKey)
		})
	}
}
