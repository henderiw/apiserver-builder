package builder

import (
	"context"
	"strings"

	"github.com/henderiw/apiserver-builder/pkg/builder/resource"
	"github.com/henderiw/apiserver-builder/pkg/builder/resource/resourcerest"
	"github.com/henderiw/apiserver-builder/pkg/builder/rest"
	"github.com/henderiw/apiserver-builder/pkg/cmd/apiserverbuilder"
	"github.com/henderiw/logger/log"
	"k8s.io/apimachinery/pkg/runtime/schema"
	regsitryrest "k8s.io/apiserver/pkg/registry/rest"
)

// WithResource registers the resource with the apiserver.
//
// If no versions of this GroupResource have already been registered, a new default handler will be registered.
// If the object implements rest.Getter, rest.Updater or rest.Creator then the provided object itself will be
// used as the rest handler for the resource type.
//
// If no versions of this GroupResource have already been registered and the object does NOT implement the rest
// interfaces, then a new etcd backed storage will be created for the object and used as the handler.
// The storage will use a DefaultStrategy, which delegates functions to the object if the object implements
// interfaces defined in the rest package.  Otherwise it will provide a default
// behavior.
//
// WithResource will automatically register the "status" subresource if the object implements the
// resource.StatusGetSetter interface.
//
// WithResource will automatically register version-specific defaulting for this GroupVersionResource
// if the object implements the resource.Defaulter interface.
//
// WithResource automatically adds the object and its list type to the known types.  If the object also declares itself
// as the storage version, the object and its list type will be added as storage versions to the SchemeBuilder as well.
// The storage version is the version accepted by the handler.
//
// If another version of the object's GroupResource has already been registered, then the resource will use the
// handler already registered for that version of the GroupResource.  Objects for this version will be converted
// to the object version which the handler accepts before the handler is invoked.
func (a *Server) WithResource(ctx context.Context, obj resource.Object) *Server {
	gvr := obj.GetGroupVersionResource()
	a.schemeBuilder.Register(resource.AddToScheme(obj))

	// reuse the storage if this resource has already been registered
	if s, found := a.storageProvider[gvr.GroupResource()]; found {
		_ = a.forGroupVersionResource(ctx, gvr, s.Get)
		return a
	}

	var parentStorageProvider rest.ResourceHandlerProvider

	defer func() {
		// automatically create status subresource if the object implements the status interface
		a.withSubResourceIfExists(ctx, obj, parentStorageProvider)
	}()

	// If the type implements it's own storage, then use that
	switch s := obj.(type) {
	case resourcerest.Creator, resourcerest.Updater, resourcerest.Getter, resourcerest.Lister:
		parentStorageProvider = rest.StaticHandlerProvider{Storage: s.(regsitryrest.Storage)}.Get
	default:
		parentStorageProvider = rest.New(obj)
	}

	_ = a.forGroupVersionResource(ctx, gvr, parentStorageProvider)

	return a
}

// WithResourceAndHandler registers a request handler for the resource rather than the default
// etcd backend storage.
//
// Note: WithResourceAndHandler should never be called after the GroupResource has already been registered with
// another version.
//
// Note: WithResourceAndHandler will NOT register the "status" subresource for the resource object.
func (r *Server) WithResourceAndHandler(ctx context.Context, obj resource.Object, sp rest.ResourceHandlerProvider) *Server {
	gvr := obj.GetGroupVersionResource()
	r.schemeBuilder.Register(resource.AddToScheme(obj))
	defer func() {
		// automatically create status subresource if the object implements the status interface
		r.withSubResourceIfExists(ctx, obj, sp)
	}()
	return r.forGroupVersionResource(ctx, gvr, sp)
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
		a.orderedGroupVersions = append(a.orderedGroupVersions, gv)
	}
	return a
}

func (a *Server) withSubResourceIfExists(
	ctx context.Context, obj resource.Object, parentStorageProvider rest.ResourceHandlerProvider) {
	//
	parentGVR := obj.GetGroupVersionResource()
	// automatically create status subresource if the object implements the status interface
	if _, ok := obj.(resource.ObjectWithStatusSubResource); ok {
		statusGVR := parentGVR.GroupVersion().WithResource(parentGVR.Resource + "/status")
		a.forGroupVersionSubResource(ctx, statusGVR, parentStorageProvider, nil)
	}
	if _, ok := obj.(resource.ObjectWithScaleSubResource); ok {
		subResourceGVR := parentGVR.GroupVersion().WithResource(parentGVR.Resource + "/scale")
		a.forGroupVersionSubResource(ctx, subResourceGVR, parentStorageProvider, nil)
	}
	if sgs, ok := obj.(resource.ObjectWithArbitrarySubResource); ok {
		for _, sub := range sgs.GetArbitrarySubResources() {
			sub := sub
			subResourceGVR := parentGVR.GroupVersion().WithResource(parentGVR.Resource + "/" + sub.SubResourceName())
			a.forGroupVersionSubResource(ctx, subResourceGVR, parentStorageProvider, rest.ParentStaticHandlerProvider{
				Storage:        sub,
				ParentProvider: parentStorageProvider,
			}.Get)
		}
	}
}

// forGroupVersionResource manually registers storage for a specific resource.
func (a *Server) forGroupVersionResource(
	ctx context.Context, gvr schema.GroupVersionResource, sp rest.ResourceHandlerProvider) *Server {
	// register the group version
	a.withGroupVersions(gvr.GroupVersion())

	// TODO: make sure folks don't register multiple storageProvider instance for the same group-resource
	// don't replace the existing instance otherwise it will chain wrapped singletonProviders when
	// fetching from the map before calling this function
	if _, found := a.storageProvider[gvr.GroupResource()]; !found {
		a.storageProvider[gvr.GroupResource()] = &singletonProvider{Provider: sp}
	}
	// add the API with its storageProvider
	apiserverbuilder.APIs[gvr] = sp
	return a
}

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
