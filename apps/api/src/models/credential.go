package models

import (
	"database/sql/driver"
	"ebs/src/types"
	"encoding/json"
	"errors"
	"log"

	"github.com/go-webauthn/webauthn/webauthn"
)

type Credential struct {
	ID         string       `gorm:"primarykey;type:text" json:"-"`
	DeviceName string       `gorm:"->;<-:create" json:"name"`
	UserID     uint         `gorm:"->;<-:create" json:"-"`
	PublicKey  string       `gorm:"->;<-:create" json:"-"`
	RawCreds   *types.JSONB `gorm:"->;<-:create;type:jsonb" json:"-"`

	Owner *User `gorm:"foreignKey:user_id" json:"-"`

	types.Timestamps
}

func (a Credential) Value() (driver.Value, error) {
	valueString, err := json.Marshal(a)
	return string(valueString), err
}
func (a *Credential) Scan(value any) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	if err := json.Unmarshal(b, &a); err != nil {
		return err
	}
	return nil
}

func (a *Credential) UnmarshalRawCredentials() (*webauthn.Credential, error) {
	b, err := json.Marshal(a.RawCreds)
	if err != nil {
		log.Printf("Could not marshal json: %s\n", err.Error())
		return nil, err
	}
	var rc webauthn.Credential
	if err := json.Unmarshal(b, &rc); err != nil {
		log.Printf("Could not unmarshal to RawCredentials: %s\n", err.Error())
		return nil, err
	}
	return &rc, nil
}
