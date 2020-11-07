package guac

import (
	"bytes"
	"encoding/base64"
	"net/http"
	"reflect"
	"strings"
)

// Restricted return 401 Unauthorized
func Restricted(w http.ResponseWriter, r *http.Request) {
	// set 401 Unauthorized
	if GetSetting().Server.Auth.Basic.Username != "" {
		w.Header().Set("WWW-Authenticate", `Basic realm="`+GetSetting().Server.Auth.Basic.Realm+`"`)
	}
	// 401 status code
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte("401 Unauthorized"))
}

// AuthManager auth methods
type AuthManager struct {
	handler    http.Handler
	handleFunc func(http.ResponseWriter, *http.Request)
}

// NewAuthManager constructor
func NewAuthManager(handler http.Handler) *AuthManager {
	return &AuthManager{
		handler: handler,
	}
}

// NewAuthManagerWithFunc constructor with func
func NewAuthManagerWithFunc(handleFunc func(http.ResponseWriter, *http.Request)) *AuthManager {
	return &AuthManager{
		handleFunc: handleFunc,
	}
}

func (s *AuthManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.handler != nil {
		Auth(w, r, s.handler.ServeHTTP)
	} else {
		Auth(w, r, s.handleFunc)
	}

}

// Auth do auth
func Auth(w http.ResponseWriter, r *http.Request, f func(http.ResponseWriter, *http.Request)) {
	if GetSetting().Server.Auth.Basic.Username == "" && GetSetting().Server.Auth.Header.Name == "" {
		f(w, r)
		return
	}
	basicAuthPrefix := "Basic "
	// get request header
	auth := r.Header.Get("Authorization")
	// if is http basic auth
	if auth != "" && strings.HasPrefix(auth, basicAuthPrefix) {
		// decode info
		payload, err := base64.StdEncoding.DecodeString(
			auth[len(basicAuthPrefix):],
		)
		if err == nil {
			pair := bytes.SplitN(payload, []byte(":"), 2)
			if len(pair) == 2 && bytes.Equal(pair[0], []byte(GetSetting().Server.Auth.Basic.Username)) &&
				bytes.Equal(pair[1], []byte(GetSetting().Server.Auth.Basic.Password)) {
				f(w, r)
				return
			}
		}
	}

	if GetSetting().Server.Auth.Header.Name != "" {
		auth := r.Header.Get(GetSetting().Server.Auth.Header.Name)
		if auth != "" {
			if len(GetSetting().Server.Auth.Header.Values) > 0 {
				if indexOf(GetSetting().Server.Auth.Header.Values, auth) > -1 {
					f(w, r)
					return
				}
			} else {
				f(w, r)
				return
			}
		}
	}
	Restricted(w, r)
}

// indexOf
func indexOf(s interface{}, elem interface{}) int {
	arrV := reflect.ValueOf(s)

	if arrV.Kind() == reflect.Slice {
		for i := 0; i < arrV.Len(); i++ {

			// XXX - panics if slice element points to an unexported struct field
			// see https://golang.org/pkg/reflect/#Value.Interface
			if arrV.Index(i).Interface() == elem {
				return i
			}
		}
	}

	return -1
}
