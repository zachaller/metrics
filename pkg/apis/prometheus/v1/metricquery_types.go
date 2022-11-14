
/*
Copyright 2022.

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

package v1

import (
	"context"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
 	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/apiserver-runtime/pkg/builder/resource"
	"sigs.k8s.io/apiserver-runtime/pkg/builder/resource/resourcestrategy"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// MetricQuery
// +k8s:openapi-gen=true
type MetricQuery struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MetricQuerySpec   `json:"spec,omitempty"`
	Status MetricQueryStatus `json:"status,omitempty"`
}

// MetricQueryList
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MetricQueryList struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []MetricQuery `json:"items"`
}

// MetricQuerySpec defines the desired state of MetricQuery
type MetricQuerySpec struct {
}

var _ resource.Object = &MetricQuery{}
var _ resourcestrategy.Validater = &MetricQuery{}

func (in *MetricQuery) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *MetricQuery) NamespaceScoped() bool {
	return false
}

func (in *MetricQuery) New() runtime.Object {
	return &MetricQuery{}
}

func (in *MetricQuery) NewList() runtime.Object {
	return &MetricQueryList{}
}

func (in *MetricQuery) GetGroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "prometheus.metrics.argoproj.io",
		Version:  "v1",
		Resource: "metricqueries",
	}
}

func (in *MetricQuery) IsStorageVersion() bool {
	return true
}

func (in *MetricQuery) Validate(ctx context.Context) field.ErrorList {
	// TODO(user): Modify it, adding your API validation here.
	return nil
}

var _ resource.ObjectList = &MetricQueryList{}

func (in *MetricQueryList) GetListMeta() *metav1.ListMeta {
	return &in.ListMeta
}
// MetricQueryStatus defines the observed state of MetricQuery
type MetricQueryStatus struct {
}

func (in MetricQueryStatus) SubResourceName() string {
	return "status"
}

// MetricQuery implements ObjectWithStatusSubResource interface.
var _ resource.ObjectWithStatusSubResource = &MetricQuery{}

func (in *MetricQuery) GetStatus() resource.StatusSubResource {
	return in.Status
}

// MetricQueryStatus{} implements StatusSubResource interface.
var _ resource.StatusSubResource = &MetricQueryStatus{}

func (in MetricQueryStatus) CopyTo(parent resource.ObjectWithStatusSubResource) {
	parent.(*MetricQuery).Status = in
}
