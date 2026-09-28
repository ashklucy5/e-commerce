package adminauth

import (
	"fmt"

	platformconfig "project.local/commerce-api/internal/platform/config"
	platformsecurity "project.local/commerce-api/internal/platform/security"
)

func NewEncryptionKeyring(
	cfg platformconfig.AdminSecurityConfig,
) (*platformsecurity.Keyring, error) {
	key, err :=
		platformsecurity.DecodeAES256KeyBase64(
			cfg.EncryptionKeyBase64,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"decode Admin encryption key: %w",
				err,
			)
	}

	keyring, err :=
		platformsecurity.NewSingleKeyKeyring(
			cfg.EncryptionKeyID,
			key,
		)
	if err != nil {
		return nil,
			fmt.Errorf(
				"create Admin encryption keyring: %w",
				err,
			)
	}

	return keyring,
		nil
}
