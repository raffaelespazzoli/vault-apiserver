package vaultstorage

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"

	"github.com/go-logr/logr"
	vault "github.com/hashicorp/vault/api"
	redhatcopv1alpha1 "github.com/redhat-cop/vault-apiserver/api/v1alpha1"
	"github.com/spf13/viper"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metainternalversion "k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	builderrest "sigs.k8s.io/apiserver-runtime/pkg/builder/rest"
)

// NewVaultStorageProvider represent a mapping between a kube object and vault resource
// rootpath is the base path for that resource
func NewVaultRoleResourceProvider(baselog logr.Logger) builderrest.ResourceHandlerProvider {

	vaultRoleResource := vaultRoleResource{
		log:          baselog.WithName("VaultRoleResource"),
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
		vaultRoleResource.vclient = client
		return &vaultRoleResource, nil
	}
}

var _ rest.Storage = &vaultRoleResource{}
var _ rest.Getter = &vaultRoleResource{}

//var _ rest.Lister = &vaultRoleResource{}
var _ rest.CreaterUpdater = &vaultRoleResource{}
var _ rest.GracefulDeleter = &vaultRoleResource{}
var _ rest.CollectionDeleter = &vaultRoleResource{}
var _ rest.Watcher = &vaultRoleResource{}

//var _ rest.StandardStorage = &vaultRoleResource{}
var _ rest.Scoper = &vaultRoleResource{}

type vaultRoleResource struct {
	vclient      *vault.Client
	log          logr.Logger
	isNamespaced bool
	newFunc      func() runtime.Object
	newListFunc  func() runtime.Object
	watchers     map[int]*jsonWatch
	muWatchers   sync.RWMutex
}

func (f *vaultRoleResource) GetLock() *sync.RWMutex {
	return &f.muWatchers
}
func (f *vaultRoleResource) GetWatchers() map[int]*jsonWatch {
	return f.watchers
}

func (f *vaultRoleResource) New() runtime.Object {
	return f.newFunc()
}

func (f *vaultRoleResource) NewList() runtime.Object {
	return f.newListFunc()
}

func (f *vaultRoleResource) NamespaceScoped() bool {
	return f.isNamespaced
}

func (f *vaultRoleResource) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions, listOptions *metainternalversion.ListOptions) (runtime.Object, error) {
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

func (f *vaultRoleResource) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
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

func (f *vaultRoleResource) get(role string) (*redhatcopv1alpha1.PolicyBinding, error) {
	secret, err := f.vclient.Logical().Read(viper.GetString(KubeAuthPathFlagName) + "/role/" + role)
	if err != nil {
		f.log.Error(err, "unable to lookup role", "path", viper.GetString(KubeAuthPathFlagName)+"/role/"+role)
		return nil, err
	}

	name, namespace, policyBindingSpec := redhatcopv1alpha1.FromVaultRole(secret.Data)

	return &redhatcopv1alpha1.PolicyBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "vault.redhatcop.redhat.io/v1alpha1",
			Kind:       "PolicyBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: *policyBindingSpec,
	}, nil
}

func (f *vaultRoleResource) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	if ri, ok := genericapirequest.RequestInfoFrom(ctx); ok {
		f.log.Info("role", "namespace", ri.Namespace, "name", ri.Name)
		return f.get(ri.Namespace + "-" + name)
	}
	return nil, errors.New("context must contain a request info")
}

func (f *vaultRoleResource) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {
	if ri, ok := genericapirequest.RequestInfoFrom(ctx); ok {
		secret, err := f.vclient.Logical().List(viper.GetString(KubeAuthPathFlagName) + "/role")
		if err != nil {
			f.log.Error(err, "unable to list roles", "path", viper.GetString(viper.GetString(KubeAuthPathFlagName)+"/role"))
			return nil, err
		}
		rolesToBeRetrieved := []string{}
		if ri.Namespace != "" {
			for _, key := range secret.Data["keys"].([]string) {
				if strings.HasPrefix(key, ri.Namespace) {
					rolesToBeRetrieved = append(rolesToBeRetrieved, key)
				}
			}
		} else {
			rolesToBeRetrieved = secret.Data["keys"].([]string)
		}
		policyBindingList := &redhatcopv1alpha1.PolicyBindingList{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "List",
			},
		}
		for _, role := range rolesToBeRetrieved {
			policyBinding, err := f.get(role)
			if err != nil {
				f.log.Error(err, "unable to retrieve", "role", role)
				return nil, err
			}
			policyBindingList.Items = append(policyBindingList.Items, *policyBinding)
		}
		return policyBindingList, nil
	}
	return nil, errors.New("context must contain a request info")
}

func (f *vaultRoleResource) create(policyBinding *redhatcopv1alpha1.PolicyBinding) (*redhatcopv1alpha1.PolicyBinding, error) {
	secret, err := f.vclient.Logical().Write(viper.GetString(KubeAuthPathFlagName)+"/role/"+policyBinding.Namespace+"-"+policyBinding.Name, policyBinding.ToVaultRole())
	if err != nil {
		f.log.Error(err, "unable to write", "role", policyBinding.ToVaultRole(), "path", KubeAuthPathFlagName+"/role/"+policyBinding.Namespace+"-"+policyBinding.Name)
		return nil, err
	}
	name, namespace, policyBindingSpec := redhatcopv1alpha1.FromVaultRole(secret.Data)
	return &redhatcopv1alpha1.PolicyBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "vault.redhatcop.redhat.io/v1alpha1",
			Kind:       "PolicyBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: *policyBindingSpec,
	}, nil
}

func (f *vaultRoleResource) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	if createValidation != nil {
		if err := createValidation(ctx, obj); err != nil {
			f.log.Error(err, "unable to validate object")
			return nil, err
		}
	}
	policyBinding, ok := obj.(*redhatcopv1alpha1.PolicyBinding)
	if !ok {
		err := errors.New("object is not of type PolicyBinding")
		f.log.Error(err, "", "obj", obj)
		return nil, err
	}

	obj, err := f.create(policyBinding)

	if err != nil {
		f.log.Error(err, "unable to create", "policyBinding", policyBinding)
	}

	f.notifyWatchers(watch.Event{
		Type:   watch.Added,
		Object: obj,
	})

	return obj, nil
}

func (f *vaultRoleResource) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	if ns, ok := genericapirequest.NamespaceFrom(ctx); ok {
		obj, err := f.Get(ctx, name, &metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return obj, true, nil
			} else {
				f.log.Error(err, "unable to lookup role")
				return nil, false, err
			}
		}
		secret, err := f.vclient.Logical().Delete(viper.GetString(KubeAuthPathFlagName) + "/role/" + ns + "-" + name)
		if err != nil {
			f.log.Error(err, "unable to delete", "role", viper.GetString(KubeAuthPathFlagName)+"/role/"+ns+"-"+name)
			return nil, false, err
		}
		name, namespace, policyBindingSpec := redhatcopv1alpha1.FromVaultRole(secret.Data)

		obj = &redhatcopv1alpha1.PolicyBinding{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "vault.redhatcop.redhat.io/v1alpha1",
				Kind:       "PolicyBinding",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: namespace,
			},
			Spec: *policyBindingSpec,
		}

		f.notifyWatchers(watch.Event{
			Type:   watch.Deleted,
			Object: obj,
		})
		return obj, true, nil

	}
	return nil, false, errors.New("context must contain a request info")
}

func (f *vaultRoleResource) notifyWatchers(ev watch.Event) {
	f.muWatchers.RLock()
	for _, w := range f.watchers {
		w.ch <- ev
	}
	f.muWatchers.RUnlock()
}

func (f *vaultRoleResource) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
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
