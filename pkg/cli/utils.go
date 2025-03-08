package cli

import (
	"fmt"
	"github.com/keybase/dbus"
	"github.com/keybase/go-keychain/secretservice"
	"reflect"
	"strings"
)

const (
	service    = "odi"
	collection = secretservice.DefaultCollection
)

func FillKeychainValues[T any](args *T) error {
	var svc *secretservice.SecretService
	var session *secretservice.Session
	var err error
	v := reflect.ValueOf(args).Elem()
	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		if f.Kind() != reflect.String {
			continue
		}
		if strings.HasPrefix(f.String(), "keychain:") {
			if svc == nil {
				svc, session, err = initSecretService()
				if err != nil {
					return fmt.Errorf("init secret service: %v", err)
				}
			}
			if session == nil {
				return fmt.Errorf("no session")
			}
			keychainElement := strings.TrimPrefix(f.String(), "keychain:")
			items, err := svc.SearchCollection(collection, secretservice.Attributes{
				"service": service,
				"element": keychainElement,
			})
			if err != nil {
				return fmt.Errorf("search keychain element: %v", err)
			}
			if len(items) < 1 {
				return fmt.Errorf("keychain element %s not found", keychainElement)
			}
			if len(items) > 1 {
				return fmt.Errorf("found more than one keychain elements for %s", keychainElement)
			}
			secretValue, err := svc.GetSecret(items[0], *session)
			if err != nil {
				return fmt.Errorf("get value from keychain: %v", err)
			}
			if !f.CanSet() {
				return fmt.Errorf("set value for field %s", v.Type().Field(i).Name)
			}
			f.SetString(string(secretValue))
		}
	}
	return nil
}

func initSecretService() (*secretservice.SecretService, *secretservice.Session, error) {
	svc, err := secretservice.NewService()
	if err != nil {
		return nil, nil, fmt.Errorf("create keychain service: %v", err)
	}
	if err := svc.Unlock([]dbus.ObjectPath{collection}); err != nil {
		return nil, nil, fmt.Errorf("unlock keychain service: %v", err)
	}
	session, err := svc.OpenSession(secretservice.AuthenticationDHAES)
	if err != nil {
		return nil, nil, fmt.Errorf("open session: %v", err)
	}
	return svc, session, nil
}
