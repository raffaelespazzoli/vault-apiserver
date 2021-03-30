/*
Copyright 2021 Red Hat Community Of Practice.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	"strconv"

	vault "github.com/hashicorp/vault/api"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/apiserver-runtime/pkg/builder/resource"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

var _ resource.Object = &SecretEngine{}
var _ resource.ObjectWithStatusSubResource = &SecretEngine{}

func (in *SecretEngine) GetGroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vault.redhatcop.redhat.io",
		Version:  "v1alpha1",
		Resource: "secretengines",
	}
}

func (in *SecretEngine) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *SecretEngine) IsStorageVersion() bool {
	return true
}

func (in *SecretEngine) NamespaceScoped() bool {
	return true
}

func (in *SecretEngine) New() runtime.Object {
	return &SecretEngine{}
}

func (in *SecretEngine) NewList() runtime.Object {
	return &SecretEngineList{}
}

func (in *SecretEngine) GetStatus() resource.StatusSubResource {
	return in.Status
}

func (in SecretEngineStatus) SubResourceName() string {
	return "status"
}

func (in SecretEngineStatus) CopyTo(parent resource.ObjectWithStatusSubResource) {
	parent.(*SecretEngine).Status = in
}

// SecretEngineSpec defines the desired state of SecretEngine
type SecretEngineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of SecretEngine. Edit secretengine_types.go to remove/update
	Mount `json:",inline"`
}

// SecretEngineStatus defines the observed state of SecretEngine
type SecretEngineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	MountStatus Mount  `json:"mountStatus,omitempty"`
	Error       string `json:"error,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// SecretEngine is the Schema for the secretengines API
type SecretEngine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecretEngineSpec   `json:"spec,omitempty"`
	Status SecretEngineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// SecretEngineList contains a list of SecretEngine
type SecretEngineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretEngine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecretEngine{}, &SecretEngineList{})
}

type Mount struct {
	Type                  string            `json:"type"`
	Description           string            `json:"description"`
	Config                MountConfig       `json:"config"`
	Local                 bool              `json:"local"`
	SealWrap              bool              `json:"seal_wrap" mapstructure:"seal_wrap"`
	ExternalEntropyAccess bool              `json:"external_entropy_access" mapstructure:"external_entropy_access"`
	Options               map[string]string `json:"options"`
}

type MountConfig struct {
	Options                   map[string]string `json:"options" mapstructure:"options"`
	DefaultLeaseTTL           string            `json:"default_lease_ttl" mapstructure:"default_lease_ttl"`
	Description               *string           `json:"description,omitempty" mapstructure:"description"`
	MaxLeaseTTL               string            `json:"max_lease_ttl" mapstructure:"max_lease_ttl"`
	ForceNoCache              bool              `json:"force_no_cache" mapstructure:"force_no_cache"`
	AuditNonHMACRequestKeys   []string          `json:"audit_non_hmac_request_keys,omitempty" mapstructure:"audit_non_hmac_request_keys"`
	AuditNonHMACResponseKeys  []string          `json:"audit_non_hmac_response_keys,omitempty" mapstructure:"audit_non_hmac_response_keys"`
	ListingVisibility         string            `json:"listing_visibility,omitempty" mapstructure:"listing_visibility"`
	PassthroughRequestHeaders []string          `json:"passthrough_request_headers,omitempty" mapstructure:"passthrough_request_headers"`
	AllowedResponseHeaders    []string          `json:"allowed_response_headers,omitempty" mapstructure:"allowed_response_headers"`
	TokenType                 string            `json:"token_type,omitempty" mapstructure:"token_type"`
}

func FromMountConfigOutput(mountConfigOutput *vault.MountConfigOutput) *MountConfig {
	return &MountConfig{
		DefaultLeaseTTL:           strconv.Itoa(mountConfigOutput.DefaultLeaseTTL),
		MaxLeaseTTL:               strconv.Itoa(mountConfigOutput.MaxLeaseTTL),
		ForceNoCache:              mountConfigOutput.ForceNoCache,
		AuditNonHMACRequestKeys:   mountConfigOutput.AuditNonHMACRequestKeys,
		AuditNonHMACResponseKeys:  mountConfigOutput.AuditNonHMACResponseKeys,
		ListingVisibility:         mountConfigOutput.ListingVisibility,
		PassthroughRequestHeaders: mountConfigOutput.PassthroughRequestHeaders,
		AllowedResponseHeaders:    mountConfigOutput.AllowedResponseHeaders,
		TokenType:                 mountConfigOutput.TokenType,
	}
}

func (mountConfig *MountConfig) getMountConfigInputFromMountConfig() *vault.MountConfigInput {
	return &vault.MountConfigInput{
		Options:                   mountConfig.Options,
		DefaultLeaseTTL:           mountConfig.DefaultLeaseTTL,
		Description:               mountConfig.Description,
		MaxLeaseTTL:               mountConfig.MaxLeaseTTL,
		ForceNoCache:              mountConfig.ForceNoCache,
		AuditNonHMACRequestKeys:   mountConfig.AuditNonHMACRequestKeys,
		AuditNonHMACResponseKeys:  mountConfig.AuditNonHMACResponseKeys,
		ListingVisibility:         mountConfig.ListingVisibility,
		PassthroughRequestHeaders: mountConfig.PassthroughRequestHeaders,
		AllowedResponseHeaders:    mountConfig.AllowedResponseHeaders,
		TokenType:                 mountConfig.TokenType,
	}
}

func FromMountOutput(mountOutput *vault.MountOutput) *Mount {
	return &Mount{
		Type:                  mountOutput.Type,
		Description:           mountOutput.Description,
		Config:                *FromMountConfigOutput(&mountOutput.Config),
		Local:                 mountOutput.Local,
		SealWrap:              mountOutput.SealWrap,
		ExternalEntropyAccess: mountOutput.ExternalEntropyAccess,
		Options:               mountOutput.Options,
	}
}

func (mount *Mount) GetMountInputFromMount() *vault.MountInput {
	return &vault.MountInput{
		Type:                  mount.Type,
		Description:           mount.Description,
		Config:                *mount.Config.getMountConfigInputFromMountConfig(),
		Local:                 mount.Local,
		SealWrap:              mount.SealWrap,
		ExternalEntropyAccess: mount.ExternalEntropyAccess,
		Options:               mount.Options,
	}
}
