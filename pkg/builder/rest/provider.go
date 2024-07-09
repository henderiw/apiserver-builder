package rest

import (
	"context"

	//"github.com/henderiw/apiserver-builder/pkg/apiserver"
	//contextutil "github.com/henderiw/apiserver-builder/pkg/util/context"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	genericregistry "k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
)

type ResourceStorageProviderFn func(ctx context.Context, scheme *runtime.Scheme, getter genericregistry.RESTOptionsGetter) (rest.Storage, error)

type SubResourceStorageProviderFn func(ctx context.Context, scheme *runtime.Scheme, store rest.Storage) (rest.Storage, error)

type StorageProvider struct {
	//Obj                                  resource.Object
	ResourceStorageProviderFn            ResourceStorageProviderFn
	StatusSubResourceStorageProviderFn   SubResourceStorageProviderFn
	ArbitrarySubresourceHandlerProviders map[string]SubResourceStorageProviderFn
}

/*
// ResourceHandlerProvider provides a request handler for a resource
type ResourceHandlerProvider = apiserver.StorageProvider

// StaticHandlerProvider returns itself as the request handler.
type StaticHandlerProvider struct { // TODO: privatize
	rest.Storage
}

// Get returns itself as the handler
func (p StaticHandlerProvider) Get(ctx context.Context, s *runtime.Scheme, g genericregistry.RESTOptionsGetter) (rest.Storage, error) {
	return p.Storage, nil
}

// ParentStaticHandlerProvider returns itself as the request handler, but with the parent
// storage plumbed in the context.
type ParentStaticHandlerProvider struct {
	rest.Storage
	ParentProvider ResourceHandlerProvider
}

// Get returns itself as the handler
func (p ParentStaticHandlerProvider) Get(ctx context.Context, s *runtime.Scheme, g genericregistry.RESTOptionsGetter) (rest.Storage, error) {
	parentStorage, err := p.ParentProvider(ctx, s, g)
	if err != nil {
		return nil, err
	}
	getter, isGetter := p.Storage.(rest.Getter)
	updater, isUpdater := p.Storage.(rest.Updater)
	switch {
	case isGetter && isUpdater:
		return parentPlumbedStorageGetterUpdaterProvider{
			getter:        getter,
			updater:       updater,
			parentStorage: parentStorage,
		}, nil
	case isGetter:
		return parentPlumbedStorageGetterProvider{
			delegate:      getter,
			parentStorage: parentStorage,
		}, nil
	}
	return p.Storage, nil
}

var _ rest.Getter = &parentPlumbedStorageGetterProvider{}

type parentPlumbedStorageGetterProvider struct {
	delegate      rest.Getter
	parentStorage rest.Storage
}

func (p parentPlumbedStorageGetterProvider) New() runtime.Object {
	return p.parentStorage.New()
}

func (p parentPlumbedStorageGetterProvider) Destroy() {}

func (p parentPlumbedStorageGetterProvider) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return p.delegate.Get(contextutil.WithParentStorage(ctx, p.parentStorage), name, options)
}

type parentPlumbedStorageGetterUpdaterProvider struct {
	getter        rest.Getter
	updater       rest.Updater
	parentStorage rest.Storage
}

func (p parentPlumbedStorageGetterUpdaterProvider) New() runtime.Object {
	return p.parentStorage.New()
}

func (p parentPlumbedStorageGetterUpdaterProvider) Destroy() {}

func (p parentPlumbedStorageGetterUpdaterProvider) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	return p.getter.Get(contextutil.WithParentStorage(ctx, p.parentStorage), name, options)
}

func (p parentPlumbedStorageGetterUpdaterProvider) Update(
	ctx context.Context,
	name string,
	objInfo rest.UpdatedObjectInfo,
	createValidation rest.ValidateObjectFunc,
	updateValidation rest.ValidateObjectUpdateFunc,
	forceAllowCreate bool,
	options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	return p.updater.Update(
		contextutil.WithParentStorage(ctx, p.parentStorage),
		name,
		objInfo,
		createValidation,
		updateValidation,
		forceAllowCreate,
		options)
}
*/
