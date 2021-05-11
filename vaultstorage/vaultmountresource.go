package vaultstorage

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	vault "github.com/hashicorp/vault/api"
	redhatcopv1alpha1 "github.com/redhat-cop/vault-apiserver/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/watch"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	builderrest "sigs.k8s.io/apiserver-runtime/pkg/builder/rest"
)

// NewVaultStorageProvider represent a mapping between a kube object and vault resource
// rootpath is the base path for that resource
func NewVaultMountStorageProvider(baselog logr.Logger) builderrest.ResourceHandlerProvider {

	vaultMountResource := vaultMountResource{
		log:          baselog.WithName("VaultMountResource"),
		isNamespaced: (&redhatcopv1alpha1.SecretEngine{}).NamespaceScoped(),
		newFunc:      (&redhatcopv1alpha1.SecretEngine{}).New,
		newListFunc:  (&redhatcopv1alpha1.SecretEngine{}).NewList,
		watchers:     make(map[int]*jsonWatch, 10),
	}

	return func(scheme *runtime.Scheme, getter generic.RESTOptionsGetter) (rest.Storage, error) {
		client, err := getClient(baselog)

		if err != nil {
			baselog.Error(err, "unable to set up vault client")
			return nil, err
		}
		vaultMountResource.vclient = client
		return &vaultMountResource, nil
	}
}

var _ rest.Storage = &vaultMountResource{}
var _ rest.Getter = &vaultMountResource{}

//var _ rest.Lister = &vaultMountResource{}
var _ rest.CreaterUpdater = &vaultMountResource{}
var _ rest.GracefulDeleter = &vaultMountResource{}
var _ rest.CollectionDeleter = &vaultMountResource{}
var _ rest.Watcher = &vaultMountResource{}

//var _ rest.StandardStorage = &vaultMountResource{}
var _ rest.Scoper = &vaultMountResource{}

type vaultMountResource struct {
	vclient      *vault.Client
	log          logr.Logger
	isNamespaced bool
	newFunc      func() runtime.Object
	newListFunc  func() runtime.Object
	watchers     map[int]*jsonWatch
	muWatchers   sync.RWMutex
}

func (f *vaultMountResource) GetLock() *sync.RWMutex {
	return &f.muWatchers
}
func (f *vaultMountResource) GetWatchers() map[int]*jsonWatch {
	return f.watchers
}

func (f *vaultMountResource) New() runtime.Object {
	return f.newFunc()
}

func (f *vaultMountResource) NewList() runtime.Object {
	return f.newListFunc()
}

func (f *vaultMountResource) NamespaceScoped() bool {
	return f.isNamespaced
}

func (f *vaultMountResource) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions, listOptions *metainternalversion.ListOptions) (runtime.Object, error) {
	objList, err := f.List(ctx, listOptions)
	if err != nil {
		f.log.Error(err, "unable to list mounts")
		return nil, err
	}
	objs, err := meta.ExtractList(objList)
	if err != nil {
		f.log.Error(err, "unable to exctract list from", "obj", objList)
		return nil, err
	}
	for i := range objs {
		obj, err := meta.Accessor(objs[i])
		if err != nil {
			f.log.Error(err, "unable to get accessor for", "obj", objs[i])
			return nil, err
		}
		_, res, err := f.Delete(ctx, obj.GetName(), deleteValidation, options)
		if err != nil && !apierrors.IsNotFound(err) || !res {
			f.log.Error(err, "unable to delete ", "mount", obj)
			return nil, err
		}
	}
	return nil, nil
}

func (f *vaultMountResource) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	obj, err := f.Get(ctx, name, &metav1.GetOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		f.log.Error(err, "unable get object to be updated")
		return nil, false, err
	}
	if apierrors.IsNotFound(err) {
		obj, err = objInfo.UpdatedObject(ctx, nil)
		if err != nil {
			f.log.Error(err, "unable to call objInfo.UpdatedObject(ctx,nil)")
			return nil, false, err
		}
	} else {
		obj, err = objInfo.UpdatedObject(ctx, obj)
		if err != nil {
			f.log.Error(err, "unable to call objInfo.UpdatedObject(ctx,pbj)")
			return nil, false, err
		}
		_, res, err := f.Delete(ctx, name, nil, &metav1.DeleteOptions{})
		if err != nil && !apierrors.IsNotFound(err) || !res {
			f.log.Error(err, "unable to delete ", "mount", obj)
			return nil, false, err
		}
	}
	_, err = f.Create(ctx, obj, nil, &metav1.CreateOptions{})
	if err != nil {
		f.log.Error(err, "unable to create ", "mount", obj)
		return nil, false, err
	}

	f.notifyWatchers(watch.Event{
		Type:   watch.Modified,
		Object: obj,
	})

	return obj, true, nil
}

func (f *vaultMountResource) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	mounts, err := f.vclient.Sys().ListMounts()
	if err != nil {
		f.log.Error(err, "unable to list mounts")
		return nil, err
	}
	if ri, ok := genericapirequest.RequestInfoFrom(ctx); ok {
		for path, mountOutput := range mounts {
			//remove trailing slash
			path = strings.Trim(path, "/")
			f.log.Info("analysing mount", "path", path, "len", len(strings.Split(path, "/")))
			f.log.Info("", "namespace", ri.Namespace, "name", ri.Name)
			if len(strings.Split(path, "/")) == 2 && strings.Split(path, "/")[0] == ri.Namespace && strings.Split(path, "/")[1] == ri.Name {
				f.log.Info("found!", "path", path)
				return &redhatcopv1alpha1.SecretEngine{
					TypeMeta: metav1.TypeMeta{
						APIVersion: "vault.redhatcop.redhat.io/v1alpha1",
						Kind:       "SecretEngine",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:              strings.Split(path, "/")[1],
						Namespace:         strings.Split(path, "/")[0],
						UID:               types.UID(mountOutput.UUID),
						ResourceVersion:   mountOutput.UUID,
						CreationTimestamp: metav1.NewTime(time.Now()),
					},
					Spec: redhatcopv1alpha1.SecretEngineSpec{
						Mount: *redhatcopv1alpha1.FromMountOutput(mountOutput),
					},
				}, nil
			}
		}
		return nil, apierrors.NewNotFound(schema.GroupResource{
			Group:    "vault.redhatcop.redhat.io",
			Resource: "secretengines",
		}, ri.Name)
	}
	return nil, errors.New("no request info in context")
}

func (f *vaultMountResource) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	mounts, err := f.vclient.Sys().ListMounts()
	if err != nil {
		f.log.Error(err, "unable to list mounts")
		return nil, err
	}
	secretEngineList := &redhatcopv1alpha1.SecretEngineList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "List",
		},
	}
	if ns, ok := genericapirequest.NamespaceFrom(ctx); ok {
		for path, mountOutput := range mounts {
			path = strings.Trim(path, "/")
			if len(strings.Split(path, "/")) != 2 || strings.Split(path, "/")[0] != ns {
				continue
			}
			secretEngineList.Items = append(secretEngineList.Items, redhatcopv1alpha1.SecretEngine{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "vault.redhatcop.redhat.io/v1alpha1",
					Kind:       "SecretEngine",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      strings.Split(path, "/")[1],
					Namespace: strings.Split(path, "/")[0],
					UID:       types.UID(mountOutput.UUID),
				},
				Spec: redhatcopv1alpha1.SecretEngineSpec{
					Mount: *redhatcopv1alpha1.FromMountOutput(mountOutput),
				},
			})
		}
	} else {
		for path, mountOutput := range mounts {
			if len(strings.Split(path, "/")) != 2 {
				continue
			}
			secretEngineList.Items = append(secretEngineList.Items, redhatcopv1alpha1.SecretEngine{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "vault.redhatcop.redhat.io/v1alpha1",
					Kind:       "SecretEngine",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      strings.Split(path, "/")[1],
					Namespace: strings.Split(path, "/")[0],
					UID:       types.UID(mountOutput.UUID),
				},
				Spec: redhatcopv1alpha1.SecretEngineSpec{
					Mount: *redhatcopv1alpha1.FromMountOutput(mountOutput),
				},
			})
		}
	}
	return secretEngineList, nil
}

func (f *vaultMountResource) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	if createValidation != nil {
		if err := createValidation(ctx, obj); err != nil {
			f.log.Error(err, "unable to validate object")
			return nil, err
		}
	}
	secretEngine, ok := obj.(*redhatcopv1alpha1.SecretEngine)
	if !ok {
		err := errors.New("object is not of type SecretEngine")
		f.log.Error(err, "", "obj", obj)
		return nil, err
	}
	err := f.vclient.Sys().Mount(strings.Join([]string{secretEngine.Namespace, secretEngine.Name}, "/"), secretEngine.Spec.GetMountInputFromMount())
	if err != nil {
		f.log.Error(err, "unable to create vault mount", "obj", obj)
		return nil, err
	}

	f.notifyWatchers(watch.Event{
		Type:   watch.Added,
		Object: obj,
	})

	return obj, nil
}

func (f *vaultMountResource) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	if ns, ok := genericapirequest.NamespaceFrom(ctx); ok {
		obj, err := f.Get(ctx, name, &metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return obj, true, nil
			} else {
				f.log.Error(err, "unable to lookup mount", "path", strings.Join([]string{ns, name}, "/"))
				return nil, false, err
			}
		}
		err = f.vclient.Sys().Unmount(strings.Join([]string{ns, name}, "/"))
		if err != nil {
			f.log.Error(err, "unable to delete mount", "path", strings.Join([]string{ns, name}, "/"))
			return nil, false, err
		}
		f.notifyWatchers(watch.Event{
			Type:   watch.Deleted,
			Object: obj,
		})
		return obj, true, nil
	}
	return nil, false, errors.New("unable to find namespace in context")
}

func (f *vaultMountResource) notifyWatchers(ev watch.Event) {
	f.muWatchers.RLock()
	for _, w := range f.watchers {
		w.ch <- ev
	}
	f.muWatchers.RUnlock()
}

func (f *vaultMountResource) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
	jw := &jsonWatch{
		id: len(f.watchers),
		f:  f,
		ch: make(chan watch.Event, 10),
	}
	// On initial watch, send all the existing objects
	list, err := f.List(ctx, options)
	if err != nil {
		return nil, err
	}

	danger := reflect.ValueOf(list).Elem()
	items := danger.FieldByName("Items")

	for i := 0; i < items.Len(); i++ {
		obj := items.Index(i).Addr().Interface().(runtime.Object)
		jw.ch <- watch.Event{
			Type:   watch.Added,
			Object: obj,
		}
	}

	f.muWatchers.Lock()
	f.watchers[jw.id] = jw
	f.muWatchers.Unlock()

	return jw, nil
}
