package vaultstorage

import (
	"sync"

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
