package keys

import (
	"context"
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"strconv"
	"strings"
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

func (svc *googleKeyService) Key(v ...int) (Key, error) {
	count := len(svc.keyVersions)
	if count == 0 {
		return nil, errors.New("key empty")
	}

	ver := count - 1
	if len(v) > 0 {
		ver = v[0] - 1
	}

	if ver < 0 {
		return nil, errors.New("invalid version")
	}

	if ver >= count {
		return nil, errors.New("key not found")
	}

	svc.RLock()
	defer svc.RUnlock()

	keyVersion := svc.keyVersions[ver]

	return &googleKey{
		CryptoKeyVersion: keyVersion,
		client:           svc.client,
	}, nil
}

func (svc *googleKeyService) Signature(data []byte, ver ...int) ([]byte, error) {
	key, err := svc.Key(ver...)
	if err != nil {
		return nil, err
	}

	return key.Signature(data)
}

func (svc *googleKeyService) Verify(data []byte, sig []byte, ver ...int) (bool, error) {
	key, err := svc.Key(ver...)
	if err != nil {
		return false, err
	}

	return key.Verify(data, sig)
}

func (svc *googleKeyService) Close() error {
	return svc.client.Close()
}

type googleKey struct {
	*kmspb.CryptoKeyVersion
	client *kms.KeyManagementClient
}

func (key *googleKey) Signature(data []byte) ([]byte, error) {
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
	resp, err := key.client.AsymmetricSign(ctx, req)
	if err != nil {
		return nil, err
	}

	if resp.Name != key.Name {
		return nil, errors.New("invalid key name")
	}

	return resp.Signature, nil
}

func (key *googleKey) Verify(data []byte, sig []byte) (bool, error) {
	ctx := context.Background()
	resp, err := key.client.GetPublicKey(ctx, &kmspb.GetPublicKeyRequest{
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

	switch key := pub.(type) {
	case ed25519.PublicKey:
		return ed25519.Verify(key, data, sig), nil

	default:
		return false, errors.New("unsupported algorithm")
	}
}

func (key *googleKey) Version() int {
	parts := strings.Split(key.Name, "/")
	if len(parts) < 10 {
		return 0
	}

	ver, err := strconv.Atoi(parts[9])
	if err != nil {
		return 0
	}

	return ver
}
