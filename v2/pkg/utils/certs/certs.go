package certs

import (
	"fmt"

	"github.com/cloudflare/cfssl/signer"
	"github.com/cloudflare/cfssl/signer/local"

	"github.com/cloudflare/cfssl/csr"
	"github.com/cloudflare/cfssl/helpers"
	"github.com/cloudflare/cfssl/initca"
	"k8s.io/apimachinery/pkg/types"
)

// CreateCertificateAuthority creates CA for self signed certificates
func CreateCertificateAuthority() ([]byte, []byte, error) {
	req := csr.CertificateRequest{
		KeyRequest: &csr.KeyRequest{
			A: "rsa",
			S: 2048,
		},
		CN: "rhmp_ca",
		Hosts: []string{
			"rhmp_ca",
		},
		CA: &csr.CAConfig{
			Expiry: "8760h",
		},
	}

	cert, _, key, err := initca.New(&req)
	if err != nil {
		return nil, nil, err
	}

	return cert, key, nil
}

// CreateCertFromCA generates certs signed with CA keys
func CreateCertFromCA(namespacedName types.NamespacedName, caCert []byte, caKey []byte) ([]byte, []byte, error) {
	parsedCaCert, err := helpers.ParseCertificatePEM(caCert)
	if err != nil {
		return nil, nil, err
	}
	parsedCaKey, err := helpers.ParsePrivateKeyPEM(caKey)
	if err != nil {
		return nil, nil, err
	}

	svcFullname := fmt.Sprintf("%s.%s.svc", namespacedName.Name, namespacedName.Namespace)
	req := csr.CertificateRequest{
		KeyRequest: &csr.KeyRequest{
			A: "rsa",
			S: 2048,
		},
		CN: svcFullname,
		Hosts: []string{
			svcFullname,
		},
	}
	certReq, key, err := csr.ParseRequest(&req)
	if err != nil {
		return nil, nil, err
	}

	csigner, err := local.NewSigner(parsedCaKey, parsedCaCert, signer.DefaultSigAlgo(parsedCaKey), nil)
	if err != nil {
		return nil, nil, err
	}

	signedCert, err := csigner.Sign(signer.SignRequest{
		Hosts:   []string{svcFullname},
		Request: string(certReq),
		Subject: &signer.Subject{
			CN: svcFullname,
		},
		Profile: svcFullname,
	})
	if err != nil {
		return nil, nil, err
	}

	return signedCert, key, nil
}
