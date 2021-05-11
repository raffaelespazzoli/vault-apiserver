package vaultstorage

import (
	"io/ioutil"
	"sync"

	"github.com/go-logr/logr"
	vault "github.com/hashicorp/vault/api"
	"github.com/spf13/viper"
	"k8s.io/apimachinery/pkg/watch"
)

var KubeAuthPathFlagName = "kubernetes-authorization-mount-path"

type kubernetesAuthInput struct {
	JWT  string `json:"jwt"`
	Role string `json:"role"`
}

type Lockable interface {
	GetLock() *sync.RWMutex
	GetWatchers() map[int]*jsonWatch
}

type jsonWatch struct {
	f  Lockable
	id int
	ch chan watch.Event
}

func (w *jsonWatch) Stop() {
	w.f.GetLock().Lock()
	delete(w.f.GetWatchers(), w.id)
	w.f.GetLock().Unlock()
}

func (w *jsonWatch) ResultChan() <-chan watch.Event {
	return w.ch
}

func getClient(setupLog logr.Logger) (*vault.Client, error) {
	client, err := vault.NewClient(vault.DefaultConfig())
	if err != nil {
		setupLog.Error(err, "unable to build vault client")
		return nil, err
	}

	token, err := ioutil.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/token")
	if err != nil {
		setupLog.Error(err, "unable to read service account token")
		return nil, err
	}

	secret, err := client.Logical().Write(viper.GetString(KubeAuthPathFlagName)+"/login", map[string]interface{}{
		"jwt":  string(token),
		"role": "vault-apiserver-dev",
	})

	if err != nil {
		setupLog.Error(err, "unable to login to vault")
		return nil, err
	}

	client.SetToken(secret.Auth.ClientToken)

	tokenWatcher, err := client.NewRenewer(&vault.RenewerInput{
		Secret: secret,
	})

	if err != nil {
		setupLog.Error(err, "unable to set up client toke watcher")
		return nil, err
	}

	go tokenWatcher.Start()

	go func() {
		for {
			select {
			case renewOutput := <-tokenWatcher.RenewCh():
				{
					client.SetToken(renewOutput.Secret.Auth.ClientToken)
				}
			case _ = <-tokenWatcher.DoneCh():
				{
					return
				}
			}
		}
	}()

	return client, nil
}
