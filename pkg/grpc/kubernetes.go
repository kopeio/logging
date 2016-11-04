package grpc

import (
	"crypto/tls"
	"fmt"
	"github.com/golang/glog"
	"golang.org/x/net/context"
	"google.golang.org/grpc/metadata"
	"net/http"
)

type KubernetesAuthorizer struct {
	AuthURL   string
	TLSConfig *tls.Config
}

func NewKubernetesAuthorizer(authURL string, tlsConfig *tls.Config) *KubernetesAuthorizer {
	k := &KubernetesAuthorizer{
		AuthURL:   authURL,
		TLSConfig: tlsConfig,
	}
	return k
}

func (k *KubernetesAuthorizer) checkUsernamePassword(username, password string) error {
	tr := &http.Transport{
		TLSClientConfig: k.TLSConfig,
	}
	client := &http.Client{
		Transport: tr,
	}
	req, err := http.NewRequest("GET", k.AuthURL, nil)
	if err != nil {
		return fmt.Errorf("error building request: %v", err)
	}
	req.SetBasicAuth(username, password)
	response, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %v", err)
	}

	if response.StatusCode == 401 {
		return InvalidAuthorization
	}
	if response.StatusCode == 200 {
		return nil
	}

	return fmt.Errorf("unexpected status code: %v", response.Status)
}

func (k *KubernetesAuthorizer) Authorize(ctx context.Context) error {
	md, ok := metadata.FromContext(ctx)
	if !ok {
		return AuthorizationRequired
	}

	usernames := md[MetadataKeyUsername]
	if len(usernames) != 1 || usernames[0] == "" {
		return AuthorizationRequired
	}
	username := usernames[0]
	passwords := md[MetadataKeyPassword]
	if len(passwords) != 1 || passwords[0] == "" {
		return AuthorizationRequired
	}
	password := passwords[0]

	err := k.checkUsernamePassword(username, password)
	if err == nil {
		return nil
	}
	if err == InvalidAuthorization {
		return err
	}

	glog.Warningf("Unexpected response from kubernetes authorization: %v", err)
	return InvalidAuthorization
}
