package rest

import (
	"k8s.io/apimachinery/pkg/runtime"
	genericregistry "k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
)

type ResourceStorageProviderFn func(scheme *runtime.Scheme, getter genericregistry.RESTOptionsGetter) (rest.Storage, error)

type SubResourceStorageProviderFn func(scheme *runtime.Scheme, store rest.Storage) (rest.Storage, error)

type StorageProvider struct {
	ResourceStorageProviderFn            ResourceStorageProviderFn
	StatusSubResourceStorageProviderFn   SubResourceStorageProviderFn
	ArbitrarySubresourceHandlerProviders map[string]SubResourceStorageProviderFn
}
