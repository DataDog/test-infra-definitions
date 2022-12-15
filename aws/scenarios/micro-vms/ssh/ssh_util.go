package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"os"

	gossh "golang.org/x/crypto/ssh"
)

func GenerateSSHKeyPair() (privateKey []byte, publicKey []byte, err error) {
	priv, err := generatePrivateKey()
	if err != nil {
		return
	}

	publicKey, err = generatePublicKey(&priv.PublicKey)
	if err != nil {
		return
	}

	privateKey = encodePrivateKeyToPEM(priv)

	return
}

func encodePrivateKeyToPEM(privateKey *rsa.PrivateKey) []byte {
	privDER := x509.MarshalPKCS1PrivateKey(privateKey)

	privBlock := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: nil,
		Bytes:   privDER,
	}

	privatePEM := pem.EncodeToMemory(&privBlock)

	return privatePEM
}

func generatePrivateKey() (*rsa.PrivateKey, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return nil, err
	}

	err = privateKey.Validate()
	if err != nil {
		return nil, err
	}

	return privateKey, nil
}

func generatePublicKey(privatekey *rsa.PublicKey) ([]byte, error) {
	publicRsaKey, err := gossh.NewPublicKey(privatekey)
	if err != nil {
		return nil, err
	}

	pubKeyBytes := gossh.MarshalAuthorizedKey(publicRsaKey)

	return pubKeyBytes, nil
}

func WriteKeyToTempFile(keyBytes []byte, targetFile string) (string, error) {
	f, err := os.CreateTemp("", targetFile)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = f.Write(keyBytes)
	if err != nil {
		return "", err
	}

	return f.Name(), nil
}
