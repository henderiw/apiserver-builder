package apiserverbuilder

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"

	"github.com/henderiw/apiserver-builder/pkg/builder/resource"
	"github.com/henderiw/apiserver-builder/pkg/builder/resource/resourcestrategy"
	"github.com/henderiw/logger/log"
	configv1alpha1 "github.com/sdcio/config-server/apis/config/v1alpha1"
	configopenapi "github.com/sdcio/config-server/apis/generated/openapi"
	//"k8s.io/apiextensions-apiserver/pkg/apiserver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/apimachinery/pkg/util/managedfields"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/version"
	genericapi "k8s.io/apiserver/pkg/endpoints"
	"k8s.io/apiserver/pkg/endpoints/openapi"
	genericregistry "k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/server"
	openapibuilder3 "k8s.io/kube-openapi/pkg/builder3"
	openapiutil "k8s.io/kube-openapi/pkg/util"
)

type StorageProvider func(ctx context.Context, s *runtime.Scheme, g genericregistry.RESTOptionsGetter) (rest.Storage, error)

var (
	// Scheme defines methods for serializing and deserializing API objects.
	Scheme = runtime.NewScheme()
	// Codecs provides methods for retrieving codecs and serializers for specific
	// versions and content types.
	Codecs = serializer.NewCodecFactory(Scheme)

	ParameterScheme = runtime.NewScheme()
	ParameterCodec  = runtime.NewParameterCodec(ParameterScheme)

	APIs                = map[schema.GroupVersionResource]StorageProvider{}
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
	c := completedConfig{
		cfg.GenericConfig.Complete(),
		&cfg.ExtraConfig,
	}

	c.GenericConfig.Version = &version.Info{
		Major: "1",
		Minor: "0",
	}

	return CompletedConfig{&c}
}

// New returns a new instance of Server from the given config.
func (c completedConfig) New(ctx context.Context) (*Server, error) {
	log := log.FromContext(ctx)
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
	log.Info("completedConfig", "length", len(apiGroups))
	log.Info("completedConfig", "apiGroups", apiGroups)
	for _, apiGroupInfo := range apiGroups {
		log.Info("completedConfig", "apiGroup", apiGroupInfo)
		log.Info("completedConfig", "PrioritizedVersions", apiGroupInfo.PrioritizedVersions)
		resourceNames := make([]string, 0)
		for _, groupVersion := range apiGroupInfo.PrioritizedVersions {
			for resource, storage := range apiGroupInfo.VersionedResourcesStorageMap[groupVersion.Version] {
				kind, err := genericapi.GetResourceKind(groupVersion, storage, apiGroupInfo.Scheme)
				if err != nil {
					return nil, err
				}
				sampleObject, err := apiGroupInfo.Scheme.New(kind)
				if err != nil {
					return nil, err
				}
				name := openapiutil.GetCanonicalTypeName(sampleObject)
				resourceNames = append(resourceNames, name)

				log.Info("completedConfig", "resource", resource)
				log.Info("completedConfig", "kind", kind)
			}
		}
		defs := configopenapi.GetOpenAPIDefinitions
		openapiconfig := server.DefaultOpenAPIV3Config(defs, openapi.NewDefinitionNamer(Scheme))

		ret := &DefinitionNamer{
			typeGroupVersionKinds: map[string]groupVersionKinds{},
		}
		schema := Scheme
		for gvk, rtype := range schema.AllKnownTypes() {
			log.Info("completedConfig", "gvk", gvk, "rtype", rtype.Name())
			newGVK := gvkConvert(gvk)
			exists := false
			for _, existingGVK := range ret.typeGroupVersionKinds[typeName(rtype)] {
				if newGVK == existingGVK {
					exists = true
					break
				}
			}
			if !exists {
				ret.typeGroupVersionKinds[typeName(rtype)] = append(ret.typeGroupVersionKinds[typeName(rtype)], newGVK)
			}
		}

		for _, gvk := range ret.typeGroupVersionKinds {
			sort.Sort(gvk)
		}
		log.Info("completedConfig", "ret", ret.typeGroupVersionKinds)

		openAPISpec, err := openapibuilder3.BuildOpenAPIDefinitionsForResources(openapiconfig, resourceNames...)
		if err != nil {
			return nil, err
		}
		for k, v := range openAPISpec {
			log.Info("completedConfig", "openAPISpec key", k)
			log.Info("completedConfig", "openAPISpec spec", v)
		}
		typeConverter, err := managedfields.NewTypeConverter(openAPISpec, false)
		if err != nil {
			return nil, err
		}
		obj := configv1alpha1.BuildConfig(metav1.ObjectMeta{Name: "test", Namespace: "test"}, configv1alpha1.ConfigSpec{}, configv1alpha1.ConfigStatus{})
		tobj, err := typeConverter.ObjectToTyped(obj)
		log.Info("completedConfig", "typedValue", tobj, "error", err)

		if err := s.GenericAPIServer.InstallAPIGroup(apiGroupInfo); err != nil {
			return nil, err
		}
	}

	return s, nil
}

// DefinitionNamer is the type to customize OpenAPI definition name.
type DefinitionNamer struct {
	typeGroupVersionKinds map[string]groupVersionKinds
}

type groupVersionKinds []v1.GroupVersionKind

func (s groupVersionKinds) Len() int {
	return len(s)
}

func (s groupVersionKinds) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s groupVersionKinds) Less(i, j int) bool {
	if s[i].Group == s[j].Group {
		if s[i].Version == s[j].Version {
			return s[i].Kind < s[j].Kind
		}
		return s[i].Version < s[j].Version
	}
	return s[i].Group < s[j].Group
}

func (s groupVersionKinds) JSON() []interface{} {
	j := []interface{}{}
	for _, gvk := range s {
		j = append(j, map[string]interface{}{
			"group":   gvk.Group,
			"version": gvk.Version,
			"kind":    gvk.Kind,
		})
	}
	return j
}

func gvkConvert(gvk schema.GroupVersionKind) v1.GroupVersionKind {
	return v1.GroupVersionKind{
		Group:   gvk.Group,
		Version: gvk.Version,
		Kind:    gvk.Kind,
	}
}

func typeName(t reflect.Type) string {
	path := t.PkgPath()
	if strings.Contains(path, "/vendor/") {
		path = path[strings.Index(path, "/vendor/")+len("/vendor/"):]
	}
	return fmt.Sprintf("%s.%s", path, t.Name())
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
		for gvr, storageProviderFunc := range APIs {
			if gvr.Group == group {
				if _, found := apis[gvr.Version]; !found {
					apis[gvr.Version] = map[string]rest.Storage{}
				}
				storage, err := storageProviderFunc(ctx, s, g)
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
				if c, ok := storage.(rest.Connecter); ok {
					optionsObj, _, _ := c.NewConnectOptions()
					if optionsObj != nil {
						ParameterScheme.AddKnownTypes(gvr.GroupVersion(), optionsObj)
						Scheme.AddKnownTypes(gvr.GroupVersion(), optionsObj)
						if _, ok := optionsObj.(resource.QueryParameterObject); ok {
							if err := ParameterScheme.AddConversionFunc(&url.Values{}, optionsObj, func(src interface{}, dest interface{}, s conversion.Scope) error {
								return dest.(resource.QueryParameterObject).ConvertFromUrlValues(src.(*url.Values))
							}); err != nil {
								return nil, err
							}
						}
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
