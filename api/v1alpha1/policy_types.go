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

var _ resource.Object = &Policy{}
var _ resource.ObjectWithStatusSubResource = &Policy{}

func (in *Policy) GetGroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "vault.redhatcop.redhat.io",
		Version:  "v1alpha1",
		Resource: "policies",
	}
}

func (in *Policy) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *Policy) IsStorageVersion() bool {
	return true
}

func (in *Policy) NamespaceScoped() bool {
	return false
}

func (in *Policy) New() runtime.Object {
	return &Policy{}
}

func (in *Policy) NewList() runtime.Object {
	return &PolicyList{}
}

func (in *Policy) GetStatus() resource.StatusSubResource {
	return in.Status
}

func (in PolicyStatus) SubResourceName() string {
	return "status"
}

func (in PolicyStatus) CopyTo(parent resource.ObjectWithStatusSubResource) {
	parent.(*Policy).Status = in
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// PolicySpec defines the desired state of Policy
type PolicySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Policy. Edit policy_types.go to remove/update
	// +kubebuilder:validation:Required
	Policy string `json:"policy,omitempty"`
}

// PolicyStatus defines the observed state of Policy
type PolicyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=policies,scope=Cluster
// Policy is the Schema for the policies API
type Policy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PolicySpec   `json:"spec,omitempty"`
	Status PolicyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PolicyList contains a list of Policy
type PolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Policy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Policy{}, &PolicyList{})
}
