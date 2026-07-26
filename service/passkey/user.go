package passkey

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/model"

	webauthn "github.com/go-webauthn/webauthn/webauthn"
)

type WebAuthnUser struct {
	user        *model.User
	credentials []*model.PasskeyCredential
}

func NewWebAuthnUser(user *model.User, credentials ...*model.PasskeyCredential) *WebAuthnUser {
	list := make([]*model.PasskeyCredential, 0, len(credentials))
	for _, credential := range credentials {
		if credential != nil {
			list = append(list, credential)
		}
	}
	return &WebAuthnUser{user: user, credentials: list}
}

func NewWebAuthnUserWithCredentials(user *model.User, credentials []*model.PasskeyCredential) *WebAuthnUser {
	return NewWebAuthnUser(user, credentials...)
}

func (u *WebAuthnUser) WebAuthnID() []byte {
	if u == nil || u.user == nil {
		return nil
	}
	return []byte(strconv.Itoa(u.user.Id))
}

func (u *WebAuthnUser) WebAuthnName() string {
	if u == nil || u.user == nil {
		return ""
	}
	name := strings.TrimSpace(u.user.Username)
	if name == "" {
		return fmt.Sprintf("user-%d", u.user.Id)
	}
	return name
}

func (u *WebAuthnUser) WebAuthnDisplayName() string {
	if u == nil || u.user == nil {
		return ""
	}
	display := strings.TrimSpace(u.user.DisplayName)
	if display != "" {
		return display
	}
	return u.WebAuthnName()
}

func (u *WebAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	if u == nil || len(u.credentials) == 0 {
		return nil
	}
	creds := make([]webauthn.Credential, 0, len(u.credentials))
	for _, credential := range u.credentials {
		creds = append(creds, credential.ToWebAuthnCredential())
	}
	return creds
}

func (u *WebAuthnUser) ModelUser() *model.User {
	if u == nil {
		return nil
	}
	return u.user
}

func (u *WebAuthnUser) PasskeyCredential() *model.PasskeyCredential {
	if u == nil || len(u.credentials) == 0 {
		return nil
	}
	return u.credentials[0]
}

func (u *WebAuthnUser) PasskeyCredentials() []*model.PasskeyCredential {
	if u == nil {
		return nil
	}
	return u.credentials
}
