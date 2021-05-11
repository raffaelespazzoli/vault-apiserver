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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/apiserver-runtime/pkg/builder/resource"
)

var _ resource.Object = &PolicyBinding{}
var _ resource.ObjectWithStatusSubResource = &PolicyBinding{}

func (in *PolicyBinding) GetGroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vault.redhatcop.redhat.io",
		Version:  "v1alpha1",
		Resource: "policybindings",
	}
}

func (in *PolicyBinding) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *PolicyBinding) IsStorageVersion() bool {
	return true
}

func (in *PolicyBinding) NamespaceScoped() bool {
	return true
}

func (in *PolicyBinding) New() runtime.Object {
	return &PolicyBinding{}
}

func (in *PolicyBinding) NewList() runtime.Object {
	return &PolicyBindingList{}
}

func (in *PolicyBinding) GetStatus() resource.StatusSubResource {
	return in.Status
}

func (in PolicyBindingStatus) SubResourceName() string {
	return "status"
}

func (in PolicyBindingStatus) CopyTo(parent resource.ObjectWithStatusSubResource) {
	parent.(*PolicyBinding).Status = in
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PolicyBindingSpec defines the desired state of PolicyBinding
type PolicyBindingSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Optional
	// +kubebuilder:default:={"default"}
	ServiceAccounts []string `json:"serviceAccounts,omitempty"`

	// +listType=set
	// +kubebuilder:validation:Required
	// kubebuilder:validation:MinItems=1
	Policies []string `json:"policies,omitempty"`

	//Audience (string: "") - Optional Audience claim to verify in the JWT.
	// +kubebuilder:validation:Optional
	Audience string `json:"audience,omitempty"`

	//TokenTTL (integer: 0 or string: "") - The incremental lifetime for generated tokens. This current value of this will be referenced at renewal time.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	TokenTTL int `json:"tokenTTL,omitempty"`

	//TokenMaxTTL (integer: 0 or string: "") - The maximum lifetime for generated tokens. This current value of this will be referenced at renewal time.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	TokenMaxTTL int `json:"tokenMaxTTL,omitempty"`

	// //token_policies (array: [] or comma-delimited string: "") - List of policies to encode onto generated tokens. Depending on the auth method, this list may be supplemented by user/group/other values.
	// TokenPolices []string `json:"token_policies,omitempty"`

	//TokenBoundCIDRs (array: [] or comma-delimited string: "") - List of CIDR blocks; if set, specifies blocks of IP addresses which can authenticate successfully, and ties the resulting token to these blocks as well.
	// +kubebuilder:validation:Optional
	TokenBoundCIDRs []string `json:"tokenBoundCIDRs,omitempty"`

	//token_explicit_max_ttl (integer: 0 or string: "") - If set, will encode an explicit max TTL onto the token. This is a hard cap even if token_ttl and token_max_ttl would otherwise allow a renewal.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	TokenExplicitMaxTTL int `json:"tokenExplicitMaxTTL,omitempty"`

	//TokenNoDefaultPolicy (bool: false) - If set, the default policy will not be set on generated tokens; otherwise it will be added to the policies set in token_policies.
	// +kubebuilder:validation:Optional
	TokenNoDefaultPolicy bool `json:"tokenNoDefaultPolicy,omitempty"`

	//TokenNumUses (integer: 0) - The maximum number of times a generated token may be used (within its lifetime); 0 means unlimited. If you require the token to have the ability to create child tokens, you will need to set this value to 0.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	TokenNumUses int `json:"tokenNumUses,omitempty"`

	//TokenPeriod (integer: 0 or string: "") - The period, if any, to set on the token.
	// +kubebuilder:validation:Optional
	// +kubebuilder:validation:Minimum=0
	TokenPeriod int `json:"tokenPeriod,omitempty"`

	//tokenType (string: "") - The type of token that should be generated. Can be service, batch, or default to use the mount's tuned default (which unless changed will be service tokens). For token store roles, there are two additional possibilities: default-service and default-batch which specify the type to return unless the client requests a different type at generation time.
	// +kubebuilder:validation:Optional
	TokenType string `json:"tokenType,omitempty"`
}

// PolicyBindingStatus defines the observed state of PolicyBinding
type PolicyBindingStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// PolicyBinding is the Schema for the policybindings API
type PolicyBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PolicyBindingSpec   `json:"spec,omitempty"`
	Status PolicyBindingStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PolicyBindingList contains a list of PolicyBinding
type PolicyBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PolicyBinding `json:"items"`
}

func init() {
	SchemeBuilder.Register(&PolicyBinding{}, &PolicyBindingList{})
}

func FromVaultRole(vaultRole map[string]interface{}) (name string, namespace string, policyBindingSpec *PolicyBindingSpec) {
	return vaultRole["name"].(string), vaultRole["bound_service_account_namespaces"].([]string)[0], &PolicyBindingSpec{
		ServiceAccounts:      vaultRole["bound_service_account_names"].([]string),
		Policies:             vaultRole["token_policies"].([]string),
		Audience:             vaultRole["audience"].(string),
		TokenTTL:             vaultRole["token_tt"].(int),
		TokenMaxTTL:          vaultRole["token_max_ttl"].(int),
		TokenBoundCIDRs:      vaultRole["token_bound_cidrs"].([]string),
		TokenExplicitMaxTTL:  vaultRole["token_explicit_max_ttl"].(int),
		TokenNoDefaultPolicy: vaultRole["token_no_default_policy"].(bool),
		TokenNumUses:         vaultRole["token_num_uses"].(int),
		TokenPeriod:          vaultRole["token_period"].(int),
		TokenType:            vaultRole["token_type"].(string),
	}
}

func (in *PolicyBinding) ToVaultRole() map[string]interface{} {
	return map[string]interface{}{
		"name":                             in.Namespace + "-" + in.Name,
		"bound_service_account_names":      in.Spec.ServiceAccounts,
		"bound_service_account_namespaces": []string{in.Namespace},
		"audience":                         in.Spec.Audience,
		"token_tt":                         in.Spec.TokenTTL,
		"token_max_ttl":                    in.Spec.TokenMaxTTL,
		"token_policies":                   in.Spec.Policies,
		"token_bound_cidrs":                in.Spec.TokenBoundCIDRs,
		"token_explicit_max_ttl":           in.Spec.TokenExplicitMaxTTL,
		"token_no_default_policy":          in.Spec.TokenNoDefaultPolicy,
		"token_num_uses":                   in.Spec.TokenNumUses,
		"token_period":                     in.Spec.TokenPeriod,
		"token_type":                       in.Spec.TokenType,
	}

}
