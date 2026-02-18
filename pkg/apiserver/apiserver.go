package apiserver

import (
	"context"
	"fmt"
	"strings"

	restbuilder "github.com/henderiw/apiserver-builder/pkg/builder/rest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/sets"
	genericregistry "k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/server"
	basecompatibility "k8s.io/component-base/compatibility"
	openapiutil "k8s.io/kube-openapi/pkg/util"
)

var (
	// Scheme defines methods for serializing and deserializing API objects.
	Scheme = runtime.NewScheme()
	// Codecs provides methods for retrieving codecs and serializers for specific
	// versions and content types.
	Codecs = serializer.NewCodecFactory(Scheme)

	ParameterScheme = runtime.NewScheme()
	ParameterCodec  = runtime.NewParameterCodec(ParameterScheme)

	APIs                = map[schema.GroupVersionResource]*restbuilder.StorageProvider{}
	GenericAPIServerFns []func(*server.GenericAPIServer) *server.GenericAPIServer
)

func init() {
	err := metav1.AddMetaToScheme(ParameterScheme)
	if err != nil {
		panic(err)
	}

	// we need to add the options to empty v1
	// TODO fix the server code to avoid this
	metav1.AddToGroupVersion(Scheme, schema.GroupVersion{Version: "v1"})

	// TODO: keep the generic API server from wanting this
	unversioned := schema.GroupVersion{Group: "", Version: "v1"}
	Scheme.AddUnversionedTypes(unversioned,
		&metav1.Status{},
		&metav1.APIVersions{},
		&metav1.APIGroupList{},
		&metav1.APIGroup{},
		&metav1.APIResourceList{},
	)
}

// ExtraConfig holds custom apiserver config
type ExtraConfig struct {
	ServerName string
}

type Config struct {
	GenericConfig *server.RecommendedConfig
	ExtraConfig   ExtraConfig
}

// Server contains state for a Kubernetes cluster master/api server.
type Server struct {
	GenericAPIServer *server.GenericAPIServer
}

type completedConfig struct {
	GenericConfig server.CompletedConfig
	ExtraConfig   *ExtraConfig
}

// CompletedConfig embeds a private pointer that cannot be instantiated outside of this package.
type CompletedConfig struct {
	*completedConfig
}

// Complete fills in any fields not set that are required to have valid data. It's mutating the receiver.
func (cfg *Config) Complete() CompletedConfig {
	cfg.GenericConfig.EffectiveVersion = basecompatibility.NewEffectiveVersionFromString("", "", "")
	c := completedConfig{
		cfg.GenericConfig.Complete(),
		&cfg.ExtraConfig,
	}

	return CompletedConfig{&c}
}

// New returns a new instance of Server from the given config.
func (c completedConfig) New(ctx context.Context) (*Server, error) {
	//log := log.FromContext(ctx)
	genericServer, err := c.GenericConfig.New(c.ExtraConfig.ServerName, server.NewEmptyDelegate())
	if err != nil {
		return nil, err
	}

	genericServer = ApplyGenericAPIServerFns(genericServer)

	s := &Server{
		GenericAPIServer: genericServer,
	}

	// Add new APIs through inserting into APIs
	apiGroups, err := BuildAPIGroupInfos(ctx, Scheme, c.GenericConfig.RESTOptionsGetter)
	if err != nil {
		return nil, err
	}

	for _, apiGroupInfo := range apiGroups {
		for _, gv := range apiGroupInfo.PrioritizedVersions {
			for resource, storage := range apiGroupInfo.VersionedResourcesStorageMap[gv.Version] {
				obj := storage.New()
				name := openapiutil.GetCanonicalTypeName(obj)
				fmt.Printf("DEBUG: resource=%s canonical=%s\n", resource, name)
			}
		}
		for gvk, t := range Scheme.AllKnownTypes() {
			if strings.Contains(gvk.Group, "sdcio") {
				fmt.Printf("DEBUG scheme: gvk=%v type=%v\n", gvk, t)
			}
		}

		if err := s.GenericAPIServer.InstallAPIGroup(apiGroupInfo); err != nil {
			return nil, err
		}
		fmt.Printf("DEBUG: StaticOpenAPISpec after install, count=%d\n", len(apiGroupInfo.StaticOpenAPISpec))
		for k := range apiGroupInfo.StaticOpenAPISpec {
			if strings.Contains(k, "config") || strings.Contains(k, "sdcio") {
				fmt.Printf("DEBUG: spec key: %s\n", k)
			}
		}
	}

	return s, nil
}

// versionedStorage wraps storage so New() returns a versioned object,
// ensuring getOpenAPIModels gets versioned canonical names matching OpenAPI defs.
type versionedStorage struct {
	rest.Storage
	newObj runtime.Object
}

func (v *versionedStorage) New() runtime.Object {
	return v.newObj.DeepCopyObject()
}

func (v *versionedStorage) NamespaceScoped() bool {
	if s, ok := v.Storage.(rest.Scoper); ok {
		return s.NamespaceScoped()
	}
	return true
}

func (v *versionedStorage) GetSingularName() string {
	if s, ok := v.Storage.(rest.SingularNameProvider); ok {
		return s.GetSingularName()
	}
	return ""
}

func (v *versionedStorage) NewList() runtime.Object {
	if s, ok := v.Storage.(rest.Lister); ok {
		return s.NewList()
	}
	return nil
}

func (v *versionedStorage) ShortNames() []string {
	if s, ok := v.Storage.(rest.ShortNamesProvider); ok {
		return s.ShortNames()
	}
	return nil
}

func (v *versionedStorage) Categories() []string {
	if s, ok := v.Storage.(rest.CategoriesProvider); ok {
		return s.Categories()
	}
	return nil
}

func (v *versionedStorage) Destroy() {
    v.Storage.Destroy()
}

func BuildAPIGroupInfos(ctx context.Context, s *runtime.Scheme, g genericregistry.RESTOptionsGetter) ([]*server.APIGroupInfo, error) {
	resourcesByGroupVersion := make(map[schema.GroupVersion]sets.Set[string])
	groups := sets.New[string]()
	for gvr := range APIs {
		groups.Insert(gvr.Group)
		if resourcesByGroupVersion[gvr.GroupVersion()] == nil {
			resourcesByGroupVersion[gvr.GroupVersion()] = sets.New[string]()
		}
		resourcesByGroupVersion[gvr.GroupVersion()].Insert(gvr.Resource)
	}
	apiGroups := []*server.APIGroupInfo{}
	for _, group := range sets.List[string](groups) {
		apis := map[string]map[string]rest.Storage{}
		for gvr, storageHandler := range APIs {
			if gvr.Group == group {
				if _, found := apis[gvr.Version]; !found {
					apis[gvr.Version] = map[string]rest.Storage{}
				}

				// register the resource store
				if storageHandler.ResourceStorageProviderFn == nil {
					return nil, fmt.Errorf("gvr %s has no storageprovider registered", gvr.String())
				}
				storage, err := storageHandler.ResourceStorageProviderFn(s, g)
				if err != nil {
					return nil, err
				}

				// Keep original storage for status/subresource providers which need *registry.Store
				originalStorage := storage

				versionedGVK := schema.GroupVersionKind{
					Group:   gvr.Group,
					Version: gvr.Version,
				}

				// Wrap versioned storage so New() returns the versioned type
				if gvr.Version != runtime.APIVersionInternal {
					internalObj := storage.New()
					versionedGVK := schema.GroupVersionKind{
						Group:   gvr.Group,
						Version: gvr.Version,
					}
					if gvks, _, err := s.ObjectKinds(internalObj); err == nil {
						for _, gvk := range gvks {
							if gvk.Version == runtime.APIVersionInternal {
								versionedGVK.Kind = gvk.Kind
								break
							}
						}
					}
					if versionedGVK.Kind != "" {
						if versionedObj, err := s.New(versionedGVK); err == nil {
							storage = &versionedStorage{Storage: storage, newObj: versionedObj}
						}
					}
				}

				apis[gvr.Version][gvr.Resource] = storage

				// ... defaulting func ...

				// Use originalStorage for status/subresource providers (need *registry.Store)
				if storageHandler.StatusSubResourceStorageProviderFn != nil {
					statusStorage, err := storageHandler.StatusSubResourceStorageProviderFn(s, originalStorage)
					if err != nil {
						return nil, err
					}
					// Wrap status storage too for versioned canonical names
					if gvr.Version != runtime.APIVersionInternal && versionedGVK.Kind != "" {
						if versionedObj, err := s.New(versionedGVK); err == nil {
							statusStorage = &versionedStorage{Storage: statusStorage, newObj: versionedObj}
						}
					}
					apis[gvr.Version][gvr.Resource+"/"+"status"] = statusStorage
				}
				for subResourcename, storageProviderFn := range storageHandler.ArbitrarySubresourceHandlerProviders {
					if storageProviderFn != nil {
						subResourceStorage, err := storageProviderFn(s, originalStorage)
						if err != nil {
							return nil, err
						}

						if gvr.Version != runtime.APIVersionInternal && versionedGVK.Kind != "" {
							if versionedObj, err := s.New(versionedGVK); err == nil {
								subResourceStorage = &versionedStorage{Storage: subResourceStorage, newObj: versionedObj}
							}
						}

						apis[gvr.Version][gvr.Resource+"/"+subResourcename] = subResourceStorage
					}
				}
			}
		}
		apiGroupInfo := server.NewDefaultAPIGroupInfo(group, Scheme, ParameterCodec, Codecs)
		apiGroupInfo.VersionedResourcesStorageMap = apis
		apiGroups = append(apiGroups, &apiGroupInfo)
	}
	return apiGroups, nil
}

func ApplyGenericAPIServerFns(in *server.GenericAPIServer) *server.GenericAPIServer {
	for i := range GenericAPIServerFns {
		in = GenericAPIServerFns[i](in)
	}
	return in
}
