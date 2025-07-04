package lib

import (
	"ebs/src/config"
	"errors"
	"log"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

var webAuthn *webauthn.WebAuthn

func InitWebAuthn(timeout time.Duration, debug bool) error {
	wconfig := &webauthn.Config{
		RPDisplayName: "Silver Elven",
		RPID:          config.API_DOMAIN,
		RPOrigins: []string{
			config.API_HOST,
			config.APP_HOST,
			"https://localhost:9090",
		},
		AuthenticatorSelection: protocol.AuthenticatorSelection{
			AuthenticatorAttachment: protocol.AuthenticatorAttachment("cross-platform"),
			RequireResidentKey:      protocol.ResidentKeyNotRequired(),
			UserVerification:        protocol.VerificationPreferred,
		},
		AttestationPreference: protocol.PreferNoAttestation,
		Debug:                 debug,
		Timeouts: webauthn.TimeoutsConfig{
			Registration: webauthn.TimeoutConfig{
				Timeout: timeout,
				Enforce: !debug,
			},
			Login: webauthn.TimeoutConfig{
				Timeout: timeout,
				Enforce: !debug,
			},
		},
	}
	wauth, err := webauthn.New(wconfig)
	if err != nil {
		log.Printf("Error initializing webauth: %s\n", err.Error())
		return err
	}
	webAuthn = wauth
	return nil
}

func GetWebAuthn() (*webauthn.WebAuthn, error) {
	if webAuthn == nil {
		return nil, errors.New("webauthn not initialized. webauthn is nil")
	}
	return webAuthn, nil
}
