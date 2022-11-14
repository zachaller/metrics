// Package filepath provides filepath storage related utilities.
package memory

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/server/storage"
	"k8s.io/apiserver/pkg/storage/storagebackend"
	"sigs.k8s.io/apiserver-runtime/pkg/builder/resource"
	builderrest "sigs.k8s.io/apiserver-runtime/pkg/builder/rest"
)

// NewMemoryStorageProvider use local host path as persistent layer storage:
//
//   - For namespaced-scoped resources: the resource will be written under the root-path in
//     the following structure:
//
//     -- (root-path) --- /namespace1/ --- resource1
//     |                |
//     |                --- resource2
//     |
//     --- /namespace2/ --- resource3
//
//   - For cluster-scoped resources, there will be no mid-layer folders for namespaces:
//
//     -- (root-path) --- resource1
//     |
//     --- resource2
//     |
//     --- resource3
//
// An example of storing example resource to local filepath will be:
//
//	builder.APIServer.
//	  WithResourceAndHandler(&v1alpha1.ExampleResource{},
//	        jsonfile.NewMemoryStorageProvider(&v1alpha1.ExampleResource{})).
//	  Build()
func NewMemoryStorageProvider(obj resource.Object) builderrest.ResourceHandlerProvider {
	return func(scheme *runtime.Scheme, getter generic.RESTOptionsGetter) (rest.Storage, error) {
		gr := obj.GetGroupVersionResource().GroupResource()
		codec, _, err := storage.NewStorageCodec(storage.StorageCodecConfig{
			StorageMediaType:  runtime.ContentTypeJSON,
			StorageSerializer: serializer.NewCodecFactory(scheme),
			StorageVersion:    scheme.PrioritizedVersionsForGroup(obj.GetGroupVersionResource().Group)[0],
			MemoryVersion:     scheme.PrioritizedVersionsForGroup(obj.GetGroupVersionResource().Group)[0],
			Config:            storagebackend.Config{}, // useless fields..
		})
		if err != nil {
			return nil, err
		}

		return NewMemoryREST(
			gr,
			codec,
			obj.NamespaceScoped(),
			obj.New,
			obj.NewList,
		), nil
	}
}
