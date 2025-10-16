package main

import (
	"crypto/tls"
	"fmt"
	"log"
)

var (
	Ciphers = map[string]uint16{
		// TLS supported up to 1.2
		"TLS_RSA_WITH_AES_128_CBC_SHA":         tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"TLS_RSA_WITH_AES_256_CBC_SHA":         tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA": tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA": tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":   tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":   tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,

		// TLS 1.2 support only
		"TLS_RSA_WITH_AES_128_GCM_SHA256":               tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_RSA_WITH_AES_256_GCM_SHA384":               tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256":       tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384":       tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256":   tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
		"TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256": tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,

		// TLS 1.3 support only
		"TLS_AES_128_GCM_SHA256":       tls.TLS_AES_128_GCM_SHA256,
		"TLS_AES_256_GCM_SHA384":       tls.TLS_AES_256_GCM_SHA384,
		"TLS_CHACHA20_POLY1305_SHA256": tls.TLS_CHACHA20_POLY1305_SHA256,

		// INSECURE CIPHERS BELOW
		// TLS supported up to 1.2
		"TLS_RSA_WITH_RC4_128_SHA":            tls.TLS_RSA_WITH_RC4_128_SHA,
		"TLS_RSA_WITH_3DES_EDE_CBC_SHA":       tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_RC4_128_SHA":    tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
		"TLS_ECDHE_RSA_WITH_RC4_128_SHA":      tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
		"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA": tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,

		// TLS 1.2 support only
		"TLS_RSA_WITH_AES_128_CBC_SHA256":         tls.TLS_RSA_WITH_AES_128_CBC_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256,
	}

	FipsCiphers = map[string]uint16{
		// see: https://datadoghq.atlassian.net/wiki/spaces/SECENG/pages/2285633911/Cryptographic+security+recommendations#Transport-Layer-Security-Protocol
		// TLS 1.2 supported
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		"TLS_RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_RSA_WITH_AES_256_GCM_SHA384,

		// TLS 1.3 supported
		"TLS_AES_128_GCM_SHA256": tls.TLS_AES_128_GCM_SHA256,
		"TLS_AES_256_GCM_SHA384": tls.TLS_AES_256_GCM_SHA384,
	}
)

func GetAvailableTLSCiphersStrings() []string {
	suites := []string{}

	for k, _ := range Ciphers {
		suites = append(suites, k)
	}

	return suites
}

func GetAvailableTLSCiphers() []uint16 {
	suites := []uint16{}

	for _, v := range Ciphers {
		suites = append(suites, v)
	}

	return suites
}

func GetCipherName(c uint16) string {

	for cipherName, cipher := range Ciphers {
		if c == cipher {
			return cipherName
		}
	}

	return ""
}

func IsCipherFIPS(cipher uint16) bool {
	for _, cipherId := range FipsCiphers {
		if cipher == cipherId {
			return true
		}
	}
	return false
}

func VerifyTLSInfo(state *tls.ConnectionState) error {
	if state != nil {
		switch state.Version {
		case tls.VersionSSL30:
			log.Println("Negotiated to Version: VersionSSL30")
		case tls.VersionTLS10:
			log.Println("Negotiated to Version: VersionTLS10")
		case tls.VersionTLS11:
			log.Println("Negotiated to Version: VersionTLS11")
		case tls.VersionTLS12:
			log.Println("Negotiated to Version: VersionTLS12")
		case tls.VersionTLS13:
			log.Println("Negotiated to Version: VersionTLS13")
		default:
			log.Println("Negotiated to Unknown TLS version")
		}

		log.Printf("Negotiated cipher suite: %v", GetCipherName(state.CipherSuite))
		if !IsCipherFIPS(state.CipherSuite) {
            return fmt.Errorf("The negotiated cipher '%v' is *NOT* FIPS compliant!", GetCipherName(state.CipherSuite))
		}
	}

	return nil
}

func ParseTLSVersion(ver string) (uint16, error) {
	switch ver {
	case "1.0":
		return tls.VersionTLS10, nil
	case "1.1":
		return tls.VersionTLS11, nil
	case "1.2":
		return tls.VersionTLS12, nil
	case "1.3":
		return tls.VersionTLS13, nil
	}
	return 0, fmt.Errorf("invalid tls version '%s' (allowed 1.0, 1.1, 1.2, 1.3)", ver)
}


func filterCiphers(ciphers []string) []uint16 {
	suites := []uint16{}

	for _, c := range ciphers {
		suite, ok := Ciphers[c]
		if !ok {
			log.Printf("Specified cipher suite '%s' unavailable. Skipping...", c)
		} else {
			suites = append(suites, suite)
		}
	}

    return suites
}

func verifyTLSVersion(tlsMin string, tlsMax string) (uint16, uint16) {
	min, err := ParseTLSVersion(tlsMin)
	if err != nil {
		panic(fmt.Errorf("invalid minimum TLS version: %w", err))
	}
	max, err := ParseTLSVersion(tlsMax)
	if err != nil {
		panic(fmt.Errorf("invalid maximum TLS version: %w", err))
	}
    return min, max
}

func displayTLSInfo(tlsMin uint16, tlsMax uint16, suites []uint16) {

    log.Printf("Selected ciphers\n-------------\n")
    for _, c := range suites {
        log.Printf("* %v", GetCipherName(c))
    }
    log.Printf("-------------\n")
	log.Printf("Allowed TLS versions: %x -- %x\n", tlsMin, tlsMax)
}
