package multiwrapper

import (
	"context"
	"errors"
	sync "sync"

	wrapping "github.com/hashicorp/go-kms-wrapping"
)

const baseEncryptor = "__base__"

var _ wrapping.Wrapper = (*MultiWrapper)(nil)

var ErrKeyNotFound = errors.New("given key ID not found")

// MultiWrapper allows multiple wrappers to be used for decryption based on key
// ID. This allows for rotation of data by allowing data to be decrypted across
// multiple (possibly derived) wrappers and encrypted with the default.
// Functions on this type will likely panic if the wrapper is not created via
// NewMultiWrapper.
type MultiWrapper struct {
	m        sync.RWMutex
	wrappers map[string]wrapping.Wrapper
}

// NewMultiWrapper creates a MultiWrapper and sets its encrypting wrapper to
// the one that is passed in. This function will panic if base is nil.
func NewMultiWrapper(base wrapping.Wrapper) *MultiWrapper {
	// For safety, no real reason this should happen
	if base.KeyID() == baseEncryptor {
		panic("invalid key ID")
	}

	ret := &MultiWrapper{
		wrappers: make(map[string]wrapping.Wrapper, 3),
	}
	ret.wrappers[baseEncryptor] = base
	ret.wrappers[base.KeyID()] = base
	return ret
}

// AddWrapper adds a wrapper to the MultiWrapper. For safety, it will refuse to
// overwrite an existing wrapper; use RemoveWrapper to remove that one first.
// The return parameter indicates if the wrapper was successfully added, that
// is, it will be false if an existing wrapper would have been overridden. If
// you want to change the encrypting wrapper, create a new MultiWrapper or call
// SetEncryptingWrapper. This function will panic if w is nil.
func (m *MultiWrapper) AddWrapper(w wrapping.Wrapper) (added bool) {
	m.m.Lock()
	defer m.m.Unlock()

	wrapper := m.wrappers[w.KeyID()]
	if wrapper != nil {
		return false
	}
	m.wrappers[w.KeyID()] = w
	return true
}

// RemoveWrapper removes a wrapper from the MultiWrapper, identified by key ID.
// It will not remove the encrypting wrapper; use SetEncryptingWrapper for
// that. Returns whether or not a wrapper was removed, which will always be
// true unless it was the base encryptor.
func (m *MultiWrapper) RemoveWrapper(keyID string) (removed bool) {
	// For safety, no real reason this should happen
	if keyID == baseEncryptor {
		panic("invalid key ID")
	}

	m.m.Lock()
	defer m.m.Unlock()

	base := m.wrappers[baseEncryptor]
	if base.KeyID() == keyID {
		// Don't allow removing the base encryptor
		return false
	}
	delete(m.wrappers, keyID)
	return true
}

// SetEncryptingWrapper resets the encrypting wrapper to the one passed in. It
// will also add the previous encrypting wrapper to the set of decrypting
// wrappers; it can then be removed via its key ID and RemoveWrapper if
// desired. It will panic if w is nil. It will return false (not successful) if
// the given key ID is already in use.
func (m *MultiWrapper) SetEncryptingWrapper(w wrapping.Wrapper) (success bool) {
	// For safety, no real reason this should happen
	if w.KeyID() == baseEncryptor {
		panic("invalid key ID")
	}

	m.m.Lock()
	defer m.m.Unlock()

	m.wrappers[baseEncryptor] = w
	m.wrappers[w.KeyID()] = w
	return true
}

// WrapperForKeyID returns the wrapper for the given keyID. Returns nil if no
// wrapper was found for the given key ID.
func (m *MultiWrapper) WrapperForKeyID(keyID string) wrapping.Wrapper {
	m.m.RLock()
	defer m.m.RUnlock()

	return m.wrappers[keyID]
}

func (m *MultiWrapper) encryptor() wrapping.Wrapper {
	m.m.RLock()
	defer m.m.RUnlock()

	wrapper := m.wrappers[baseEncryptor]
	if wrapper == nil {
		panic("no base encryptor found")
	}
	return wrapper
}

func (m *MultiWrapper) Type() string {
	return wrapping.MultiWrapper
}

// KeyID returns the KeyID of the current encryptor
func (m *MultiWrapper) KeyID() string {
	return m.encryptor().KeyID()
}

// HMACKeyID returns the HMACKeyID of the current encryptor
func (m *MultiWrapper) HMACKeyID() string {
	return m.encryptor().HMACKeyID()
}

// This does nothing; it's up to the user to initialize and finalize any given
// wrapper
func (m *MultiWrapper) Init(context.Context) error {
	return nil
}

// This does nothing; it's up to the user to initialize and finalize any given
// wrapper
func (m *MultiWrapper) Finalize(context.Context) error {
	return nil
}

// Encrypt encrypts using the current encryptor
func (m *MultiWrapper) Encrypt(ctx context.Context, pt []byte, aad []byte) (*wrapping.EncryptedBlobInfo, error) {
	return m.encryptor().Encrypt(ctx, pt, aad)
}

// Decrypt will use the embedded KeyID in the encrypted blob info to select
// which wrapper to use for decryption. If there is no key info it will attempt
// decryption with the current encryptor. It will return an ErrKeyNotFound if
// it cannot find a suitable key.
func (m *MultiWrapper) Decrypt(ctx context.Context, ct *wrapping.EncryptedBlobInfo, aad []byte) ([]byte, error) {
	if ct.KeyInfo == nil {
		enc := m.encryptor()
		return enc.Decrypt(ctx, ct, aad)
	}

	wrapper := m.WrapperForKeyID(ct.KeyInfo.KeyID)
	if wrapper == nil {
		return nil, ErrKeyNotFound
	}
	return wrapper.Decrypt(ctx, ct, aad)
}
