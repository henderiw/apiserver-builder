package rest

/*
// New returns a new etcd backed request handler for the resource.
func NewEtcdProviderFn(obj resource.Object) ResourceStorageProviderFn {
	return func(ctx context.Context, scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (rest.Storage, error) {
		strategy := &DefaultStrategy{
			Object:         obj,
			ObjectTyper:    scheme,
			TableConvertor: rest.NewDefaultTableConvertor(obj.GetGroupVersionResource().GroupResource()),
		}
		return newStore(scheme, obj, strategy, optsGetter, nil)
	}
}

// New returns a new etcd backed request handler for the resource.
func NewEtcdStatusProviderFn(obj resource.Object, sp rest.ResourceStorageProviderFn) SubResourceStorageProviderFn {
	return func(ctx context.Context, scheme *runtime.Scheme, store rest.Storage) (rest.Storage, error) {
		strategy := &DefaultStatusStrategy{
			Object:        obj,
			ObjectTyper:   scheme,
			NameGenerator: names.SimpleNameGenerator,
		}
		return newStatusStore(scheme, obj, strategy, store)
	}
}
*/
/*
// NewWithStrategy returns a new etcd backed request handler using the provided Strategy.
func NewWithStrategy(obj resource.Object, s Strategy) ResourceHandlerProvider {
	return func(ctx context.Context, scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (rest.Storage, error) {
		gvr := obj.GetGroupVersionResource()
		return newStore(scheme, obj.New, obj.NewList, gvr, s, optsGetter, nil)
	}
}

// NewWithFn returns a new etcd backed request handler, applying the StoreFn to the Store.
func NewWithFn(obj resource.Object, fn StoreFn) ResourceHandlerProvider {
	return func(ctx context.Context, scheme *runtime.Scheme, optsGetter generic.RESTOptionsGetter) (rest.Storage, error) {
		gvr := obj.GetGroupVersionResource()
		s := &DefaultStrategy{
			Object:         obj,
			ObjectTyper:    scheme,
			TableConvertor: rest.NewDefaultTableConvertor(gvr.GroupResource()),
		}
		return newStore(scheme, obj.New, obj.NewList, gvr, s, optsGetter, fn)
	}
}







// SubResourceStorageFn is a function that returns objects required to register a subresource into an apiserver
// path is the subresource path from the parent (e.g. "scale"), parent is the resource the subresource
// is under (e.g. &v1.Deployment{}), request is the subresource request (e.g. &Scale{}), storage is
// the storage implementation that handles the requests.
// A SubResourceStorageFn can be used with builder.APIServer.WithSubResourceAndStorageProvider(fn())
type SubResourceStorageFn func() (path string, parent resource.Object, request resource.Object, storage ResourceHandlerProvider)

// ResourceStorageFn is a function that returns the objects required to register a resource into an apiserver.
// request is the resource type (e.g. &v1.Deployment{}), storage is the storage implementation that handles
// the requests.
// A ResourceFn can be used with builder.APIServer.WithResourceAndStorageProvider(fn())
type ResourceStorageFn func() (request resource.Object, storage ResourceHandlerProvider)
*/

/*
// newStore returns a RESTStorage object that will work against API services.
func newStore(
	scheme *runtime.Scheme,
	obj resource.Object,
	s Strategy,
	optsGetter generic.RESTOptionsGetter,
	fn StoreFn,
) (*genericregistry.Store, error) {
	store := &genericregistry.Store{
		NewFunc:                  obj.New,
		NewListFunc:              obj.NewList,
		PredicateFunc:            s.Match,
		DefaultQualifiedResource: gvr.GroupResource(),
		TableConvertor:           s,
		CreateStrategy:           s,
		UpdateStrategy:           s,
		DeleteStrategy:           s,
		StorageVersioner:         gvr.GroupVersion(),
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if fn != nil {
		fn(scheme, store, options)
	}
	if err := store.CompleteWithOptions(options); err != nil {
		return nil, err
	}
	return store, nil
}

// StoreFn defines a function which modifies the Store before it is initialized.
type StoreFn func(*runtime.Scheme, *genericregistry.Store, *generic.StoreOptions)



func newStatusStore(
	scheme *runtime.Scheme,
	single, list func() runtime.Object,
	gvr schema.GroupVersionResource,
	s Strategy, optsGetter generic.RESTOptionsGetter, fn StoreFn) (*genericregistry.Store, error) {
	store := &genericregistry.Store{
		NewFunc:                  single,
		NewListFunc:              list,
		PredicateFunc:            s.Match,
		DefaultQualifiedResource: gvr.GroupResource(),
		TableConvertor:           s,
		CreateStrategy:           s,
		UpdateStrategy:           s,
		DeleteStrategy:           s,
		StorageVersioner:         gvr.GroupVersion(),
	}

	options := &generic.StoreOptions{RESTOptions: optsGetter, AttrFunc: GetAttrs}
	if fn != nil {
		fn(scheme, store, options)
	}
	if err := store.CompleteWithOptions(options); err != nil {
		return nil, err
	}
	return store, nil
}
*/
