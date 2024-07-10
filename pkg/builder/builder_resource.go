package builder

import (
	"github.com/henderiw/apiserver-builder/pkg/apiserver"
	"github.com/henderiw/apiserver-builder/pkg/builder/resource"
	"github.com/henderiw/apiserver-builder/pkg/builder/rest"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime"
)

/*
func (r *Server) WithResource(obj resource.Object) *Server {
	gvr := obj.GetGroupVersionResource()
	r.schemeBuilder.Register(resource.AddToScheme(obj))

	var sh rest.ResourceStorageHandler
	sh.ResourceStorageProviderFn = rest.NewEtcdProviderFn(obj)

	if _, ok := obj.(resource.ObjectWithStatusSubResource); ok {
		sh.StatusSubResourceStorageProviderFn = rest.NewEtcdStatusProviderFn(obj, sh.ResourceStorageProviderFn)
	}

	//

	return r.forGroupVersionResource(gvr, sh)
}
*/

// WithResourceAndHandler registers a request handler for the resource rather than the default
// etcd backend storage.
//
// Note: WithResourceAndHandler should never be called after the GroupResource has already been registered with
// another version.
//
// Note: WithResourceAndHandler will NOT register the "status" subresource for the resource object.
func (r *Server) WithResourceAndHandler(obj resource.Object, sp rest.StorageProvider) *Server {
	gvr := obj.GetGroupVersionResource()
	r.schemeBuilder.Register(resource.AddToScheme(obj))
	return r.forGroupVersionResource(gvr, sp)
}

// WithSchemeInstallers registers functions to install resource types into the Scheme.
func (a *Server) withGroupVersions(
	versions ...schema.GroupVersion) *Server {
	//
	if a.groupVersions == nil {
		a.groupVersions = map[schema.GroupVersion]bool{}
	}
	for _, gv := range versions {
		if _, found := a.groupVersions[gv]; found {
			continue
		}
		a.groupVersions[gv] = true
		if gv.Version == runtime.APIVersionInternal {
			continue
		}
		a.orderedGroupVersions = append(a.orderedGroupVersions, gv)
	}
	return a
}

// forGroupVersionResource manually registers storage for a specific resource.
func (a *Server) forGroupVersionResource(gvr schema.GroupVersionResource, sp rest.StorageProvider) *Server {
	// register the group version
	a.withGroupVersions(gvr.GroupVersion())

	// TODO: make sure folks don't register multiple storageProvider instance for the same group-resource
	// don't replace the existing instance otherwise it will chain wrapped singletonProviders when
	// fetching from the map before calling this function
	if _, found := a.storageProvider[gvr.GroupResource()]; !found {
		a.storageProvider[gvr.GroupResource()] = &singletonProvider{Provider: sp}
	}
	// add the API with its storageProvider
	apiserver.APIs[gvr] = sp
	return a
}

/*
// forGroupVersionSubResource manually registers storageProvider for a specific subresource.
func (a *Server) forGroupVersionSubResource(
	ctx context.Context, gvr schema.GroupVersionResource, parentProvider rest.ResourceHandlerProvider, subResourceProvider rest.ResourceHandlerProvider) {
	log := log.FromContext(ctx)
	isSubResource := strings.Contains(gvr.Resource, "/")
	if !isSubResource {
		log.Error("Expected status subresource but received", "group", gvr.Group, "version", gvr.Version, "resource", gvr.Resource)
	}

	// add the API with its storageProvider for subresource
	apiserverbuilder.APIs[gvr] = (&subResourceStorageProvider{
		subResourceGVR:             gvr,
		parentStorageProvider:      parentProvider,
		subResourceStorageProvider: subResourceProvider,
	}).Get
}
*/
