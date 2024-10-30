package keys

type Service interface {
	Signature(data []byte, ver ...int) ([]byte, error)
	Verify(data, signature []byte, ver ...int) (bool, error)
	Close() error
}
