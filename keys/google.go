package keys

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"sync"

	"cloud.google.com/go/kms/apiv1/kmspb"
	"google.golang.org/api/iterator"

	kms "cloud.google.com/go/kms/apiv1"

	"github.com/flarexio/wallet/conf"
)

func NewGoogleKeysService(cfg conf.GoogleKeyConfig) (Service, error) {
	ctx := context.Background()
	client, err := kms.NewKeyManagementClient(ctx)
	if err != nil {
		return nil, err
	}

	req := &kmspb.ListCryptoKeyVersionsRequest{
		Parent: cfg.Path(),
	}

	keyVersions := make([]*kmspb.CryptoKeyVersion, 0)

	it := client.ListCryptoKeyVersions(ctx, req)
	for {
		version, err := it.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		keyVersions = append(keyVersions, version)
	}

	return &googleKeyService{
		client:      client,
		keyVersions: keyVersions,
	}, nil
}

type googleKeyService struct {
	client      *kms.KeyManagementClient
	keyVersions []*kmspb.CryptoKeyVersion
	sync.RWMutex
}

func (svc *googleKeyService) FindKeyVersion(v ...int) (*kmspb.CryptoKeyVersion, error) {
	count := len(svc.keyVersions)
	if count == 0 {
		return nil, errors.New("key empty")
	}

	ver := count - 1
	if len(v) > 0 {
		ver = v[0]
	}

	if ver > count {
		return nil, errors.New("key version not found")
	}

	svc.RLock()
	defer svc.RUnlock()

	return svc.keyVersions[ver], nil
}

func (svc *googleKeyService) Signature(data []byte, ver ...int) ([]byte, error) {
	key, err := svc.FindKeyVersion(ver...)
	if err != nil {
		return nil, err
	}

	var req *kmspb.AsymmetricSignRequest
	switch key.Algorithm {
	case kmspb.CryptoKeyVersion_EC_SIGN_ED25519:
		req = &kmspb.AsymmetricSignRequest{
			Name: key.Name,
			Data: data,
		}

	default:
		return nil, errors.New("unsupported algorithm")
	}

	ctx := context.Background()
	resp, err := svc.client.AsymmetricSign(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Name != key.Name {
		return nil, errors.New("invalid key name")
	}

	return resp.Signature, nil
}

func (svc *googleKeyService) Verify(data, signature []byte, ver ...int) (bool, error) {
	key, err := svc.FindKeyVersion(ver...)
	if err != nil {
		return false, err
	}

	ctx := context.Background()
	resp, err := svc.client.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{
		Name: key.Name,
	})
	if err != nil {
		return false, err
	}

	block, _ := pem.Decode([]byte(resp.Pem))
	if block == nil || block.Type != "PUBLIC KEY" {
		return false, errors.New("invalid public key")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return false, err
	}

	switch publicKey := pub.(type) {
	case ed25519.PublicKey:
		return ed25519.Verify(publicKey, data, signature), nil

	default:
		return false, errors.New("unsupported algorithm")
	}
}

func (svc *googleKeyService) Close() error {
	return svc.client.Close()
}
