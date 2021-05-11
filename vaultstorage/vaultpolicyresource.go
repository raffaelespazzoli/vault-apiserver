package vaultstorage

import (
	"context"
	"errors"
	"reflect"
	"sync"

	"github.com/go-logr/logr"
	vault "github.com/hashicorp/vault/api"
	redhatcopv1alpha1 "github.com/redhat-cop/vault-apiserver/api/v1alpha1"
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
func NewVaultPolicyResourceProvider(client *vault.Client, baselog logr.Logger) builderrest.ResourceHandlerProvider {
	return func(scheme *runtime.Scheme, getter generic.RESTOptionsGetter) (rest.Storage, error) {
		return &vaultMountResource{
			vclient:      client,
			log:          baselog.WithName("VaultRoleResource"),
			isNamespaced: (&redhatcopv1alpha1.SecretEngine{}).NamespaceScoped(),
			newFunc:      (&redhatcopv1alpha1.SecretEngine{}).New,
			newListFunc:  (&redhatcopv1alpha1.SecretEngine{}).NewList,
			watchers:     make(map[int]*jsonWatch, 10),
		}, nil
	}
}

var _ rest.Storage = &vaultPolicyResource{}
var _ rest.Getter = &vaultPolicyResource{}

//var _ rest.Lister = &vaultRoleResource{}
var _ rest.CreaterUpdater = &vaultPolicyResource{}
var _ rest.GracefulDeleter = &vaultPolicyResource{}
var _ rest.CollectionDeleter = &vaultPolicyResource{}
var _ rest.Watcher = &vaultPolicyResource{}

//var _ rest.StandardStorage = &vaultRoleResource{}
var _ rest.Scoper = &vaultPolicyResource{}

type vaultPolicyResource struct {
	vclient      *vault.Client
	log          logr.Logger
	isNamespaced bool
	newFunc      func() runtime.Object
	newListFunc  func() runtime.Object
	watchers     map[int]*jsonWatch
	muWatchers   sync.RWMutex
}

func (f *vaultPolicyResource) GetLock() *sync.RWMutex {
	return &f.muWatchers
}
func (f *vaultPolicyResource) GetWatchers() map[int]*jsonWatch {
	return f.watchers
}

func (f *vaultPolicyResource) New() runtime.Object {
	return f.newFunc()
}

func (f *vaultPolicyResource) NewList() runtime.Object {
	return f.newListFunc()
}

func (f *vaultPolicyResource) NamespaceScoped() bool {
	return f.isNamespaced
}

func (f *vaultPolicyResource) DeleteCollection(ctx context.Context, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions, listOptions *metainternalversion.ListOptions) (runtime.Object, error) {
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

func (f *vaultPolicyResource) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
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

func (f *vaultPolicyResource) get(policy string) (*redhatcopv1alpha1.Policy, error) {
	policyText, err := f.vclient.Sys().GetPolicy(policy)

	if err != nil {
		f.log.Error(err, "unable to lookup", "policy", policy)
		return nil, err
	}

	return &redhatcopv1alpha1.Policy{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "vault.redhatcop.redhat.io/v1alpha1",
			Kind:       "Policy",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: policy,
		},
		Spec: redhatcopv1alpha1.PolicySpec{
			Policy: policyText,
		},
	}, nil
}

func (f *vaultPolicyResource) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	if ri, ok := genericapirequest.RequestInfoFrom(ctx); ok {
		f.log.Info("role", "name", ri.Name)
		return f.get(name)
	}
	return nil, errors.New("context must contain a request info")
}

func (f *vaultPolicyResource) List(ctx context.Context, options *metainternalversion.ListOptions) (runtime.Object, error) {

	policies, err := f.vclient.Sys().ListPolicies()
	if err != nil {
		f.log.Error(err, "unable to list policies")
		return nil, err
	}
	policyList := &redhatcopv1alpha1.PolicyList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "List",
		},
	}
	for _, policy := range policies {
		policyObject, err := f.get(policy)
		if err != nil {
			f.log.Error(err, "unable to retrieve", "policy", policy)
			return nil, err
		}
		policyList.Items = append(policyList.Items, *policyObject)
	}
	return policyList, nil
}

func (f *vaultPolicyResource) create(policy *redhatcopv1alpha1.Policy) (*redhatcopv1alpha1.Policy, error) {
	err := f.vclient.Sys().PutPolicy(policy.Name, policy.Spec.Policy)
	if err != nil {
		f.log.Error(err, "unable to write", "policy", policy.Name, "content", policy.Spec.Policy)
		return nil, err
	}
	return policy, nil
}

func (f *vaultPolicyResource) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	if createValidation != nil {
		if err := createValidation(ctx, obj); err != nil {
			f.log.Error(err, "unable to validate object")
			return nil, err
		}
	}
	policy, ok := obj.(*redhatcopv1alpha1.Policy)
	if !ok {
		err := errors.New("object is not of type Policy")
		f.log.Error(err, "", "obj", obj)
		return nil, err
	}

	obj, err := f.create(policy)

	if err != nil {
		f.log.Error(err, "unable to create", "policy", policy)
	}

	f.notifyWatchers(watch.Event{
		Type:   watch.Added,
		Object: obj,
	})

	return obj, nil
}

func (f *vaultPolicyResource) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {

	obj, err := f.Get(ctx, name, &metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return obj, true, nil
		} else {
			f.log.Error(err, "unable to lookup policy")
			return nil, false, err
		}
	}
	err = f.vclient.Sys().DeletePolicy(name)
	if err != nil {
		f.log.Error(err, "unable to delete", "policy", name)
		return nil, false, err
	}

	f.notifyWatchers(watch.Event{
		Type:   watch.Deleted,
		Object: obj,
	})
	return obj, true, nil
}

func (f *vaultPolicyResource) notifyWatchers(ev watch.Event) {
	f.muWatchers.RLock()
	for _, w := range f.watchers {
		w.ch <- ev
	}
	f.muWatchers.RUnlock()
}

func (f *vaultPolicyResource) Watch(ctx context.Context, options *metainternalversion.ListOptions) (watch.Interface, error) {
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
