package apiserver

import (
	"context"
	"fmt"

	"github.com/henderiw/apiserver-builder/pkg/builder/resource/resourcestrategy"
	restbuilder "github.com/henderiw/apiserver-builder/pkg/builder/rest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/sets"
	genericregistry "k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/server"
	utilversion "k8s.io/apiserver/pkg/util/version"
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
	metav1.AddMetaToScheme(ParameterScheme)

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
	cfg.GenericConfig.EffectiveVersion = utilversion.NewEffectiveVersion("")
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
		if err := s.GenericAPIServer.InstallAPIGroup(apiGroupInfo); err != nil {
			return nil, err
		}
	}

	return s, nil
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

				apis[gvr.Version][gvr.Resource] = storage
				// add the defaulting function for this version to the scheme
				if _, ok := storage.(resourcestrategy.Defaulter); ok {
					if obj, ok := storage.(runtime.Object); ok {
						s.AddTypeDefaultingFunc(obj, func(obj interface{}) {
							obj.(resourcestrategy.Defaulter).Default()
						})
					}
				}
				// register the status subresource store if exists
				if storageHandler.StatusSubResourceStorageProviderFn != nil {
					statusstorage, err := storageHandler.StatusSubResourceStorageProviderFn(s, storage)
					if err != nil {
						return nil, err
					}
					apis[gvr.Version][gvr.Resource+"/"+"status"] = statusstorage

				}
				// register the arbitray subresource stores if exists
				for subResourcename, storageProviderFn := range storageHandler.ArbitrarySubresourceHandlerProviders {
					if storageProviderFn != nil {
						subResourceStorage, err := storageProviderFn(s, storage)
						if err != nil {
							return nil, err
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
