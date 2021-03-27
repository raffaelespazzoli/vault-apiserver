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

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

var _ resource.Object = &SecretEngine{}
var _ resource.ObjectWithStatusSubResource = &SecretEngine{}

func (in *SecretEngine) GetGroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "redhatcop.redhat.io",
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
	Foo string `json:"foo,omitempty"`
}

// SecretEngineStatus defines the observed state of SecretEngine
type SecretEngineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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
