package keys

import "crypto"

type Service interface {
	Key(ver ...int) (Key, error)
	Signature(data []byte, ver ...int) ([]byte, error)
	Verify(data []byte, sig []byte, ver ...int) (bool, error)
	Close() error
}

type Key interface {
	crypto.Signer
	Signature(data []byte) ([]byte, error)
	Verify(data []byte, sig []byte) (bool, error)
	Version() int
}
