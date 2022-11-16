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

// MetricQueryRun
// +k8s:openapi-gen=true
type MetricQueryRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec MetricQueryRunSpec `json:"spec,omitempty"`
}

// MetricQueryRunList
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type MetricQueryRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []MetricQueryRun `json:"items"`
}

// MetricQueryRunSpec defines the desired state of MetricQueryRun
type MetricQueryRunSpec struct {
	Results []Result `json:"results,omitempty" protobuf:"bytes,1,opt,name=results"`
}

type Result struct {
	Name   string `json:"name,omitempty" protobuf:"bytes,1,opt,name=name"`
	Result string `json:"result,omitempty" protobuf:"bytes,2,opt,name=result"`
}

var _ resource.Object = &MetricQueryRun{}
var _ resourcestrategy.Validater = &MetricQueryRun{}

func (in *MetricQueryRun) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *MetricQueryRun) NamespaceScoped() bool {
	return true
}

func (in *MetricQueryRun) New() runtime.Object {
	return &MetricQueryRun{}
}

func (in *MetricQueryRun) NewList() runtime.Object {
	return &MetricQueryRunList{}
}

func (in *MetricQueryRun) GetGroupVersionResource() schema.GroupVersionResource {
	return schema.GroupVersionResource{
		Group:    "prometheus.metrics.argoproj.io",
		Version:  "v1",
		Resource: "metricqueryruns",
	}
}

func (in *MetricQueryRun) IsStorageVersion() bool {
	return true
}

func (in *MetricQueryRun) Validate(ctx context.Context) field.ErrorList {
	// TODO(user): Modify it, adding your API validation here.
	return nil
}

var _ resource.ObjectList = &MetricQueryRunList{}

func (in *MetricQueryRunList) GetListMeta() *metav1.ListMeta {
	return &in.ListMeta
}

//func (in *MetricQueryRun) GetObjectKind() schema.ObjectKind {
//	return in.GetObjectKind()
//}
