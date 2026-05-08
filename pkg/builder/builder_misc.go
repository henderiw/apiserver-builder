package builder

import (
	"strings"

	"github.com/henderiw/apiserver-builder/pkg/apiserver"
	"github.com/henderiw/apiserver-builder/pkg/cmd/apiserverbuilder/options"
	apiextensionsopenapi "k8s.io/apiextensions-apiserver/pkg/generated/openapi"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/server"
	scheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	openapicommon "k8s.io/kube-openapi/pkg/common"
	spec "k8s.io/kube-openapi/pkg/validation/spec"
)

// goImportPathToReverseDNS converts a Go import path to a reverse-DNS OpenAPI
// component name — the same convention Kubernetes uses for io.k8s.* types.
//
//	github.com/sdcio/config-server/apis/config/v1alpha1.TargetStatus
//	→ com.github.sdcio.config-server.apis.config.v1alpha1.TargetStatus
//
//	k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta
//	→ io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta
//
// The result has no "/" or "~", so kube-openapi's EscapeJsonPointer() is a
// no-op: $ref values equal component keys exactly, enabling raw (non-decoded)
// lookups such as those performed by structured-merge-diff/v6's TypeConverter.
func goImportPathToReverseDNS(goPath string) string {
	// Split at the last non-qualified segment (type name after last ".")
	// goPath format: "pkg/path.TypeName"
	slash := strings.Index(goPath, "/")
	if slash == -1 {
		// No host segment — stdlib or simple name; return as-is.
		return goPath
	}
	host := goPath[:slash]
	rest := strings.ReplaceAll(goPath[slash+1:], "/", ".")

	parts := strings.Split(host, ".")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, ".") + "." + rest
}

// WithOpenAPIDefinitions registers OpenAPI definitions for the API server.
//
// # Naming strategy
//
// structured-merge-diff/v6 TypeConverter performs a RAW (non-JSON-Pointer-
// decoded) lookup when resolving $ref component names.  This means $ref tokens
// and component map keys must be byte-for-byte identical.
//
// kube-openapi builds $ref strings as:
//
//	"#/components/schemas/" + EscapeJsonPointer(GetDefinitionName(goPath))
//
// EscapeJsonPointer converts "/" → "~1" and "~" → "~0".
// If GetDefinitionName returns a name that already contains "~1" (old-style
// encoding), EscapeJsonPointer double-encodes it to "~01", making $ref ≠ key.
//
// Fix: GetDefinitionName returns clean reverse-DNS names ("com.github.sdcio…")
// that contain neither "/" nor "~".  EscapeJsonPointer is then a no-op, so
// $ref == component key == what the TypeConverter looks up.  kubectl also
// resolves these $refs correctly without any JSON Pointer decoding.
//
// # Alias entries
//
// getResourceNamesForGroup (inside GenericAPIServer.getOpenAPIModels) calls
// GetDefinitionName on each registered Go type to get its "canonical" name,
// then passes those names to BuildOpenAPIDefinitionsForResources, which looks
// them up as keys in the GetDefinitions map.  Since the map is keyed by Go
// import paths ("github.com/…/Target") but the canonical names are now
// reverse-DNS ("com.github.sdcio…Target"), we add reverse-DNS alias entries
// so the lookups succeed.
func (r *Server) WithOpenAPIDefinitions(
	name, version string,
	defs openapicommon.GetOpenAPIDefinitions) *Server {

	// STARTUP TRACE — this line in the server log confirms the new binary is running.
	klog.Infof("apiserver-builder/WithOpenAPIDefinitions: %q %q (reverse-dns-aliases build)", name, version)

	namer := openapinamer.NewDefinitionNamer(apiserver.Scheme, scheme.Scheme)

	// getDefinitionName converts any Go import path to a stable reverse-DNS name.
	// GVK extensions (x-kubernetes-group-version-kind) are preserved so the
	// TypeConverter can map GVK → schema name.
	getDefinitionName := func(n string) (string, spec.Extensions) {
		resolved, ext := namer.GetDefinitionName(n)
		// The namer returns names unchanged; Go import paths contain "/".
		// Old-style fallback encoding produces "~1".  Both trigger the transform.
		if strings.ContainsAny(resolved, "/~") {
			return goImportPathToReverseDNS(n), ext // preserve ext (GVK annotations)
		}
		return resolved, ext
	}

	mergedDefs := func(ref openapicommon.ReferenceCallback) map[string]openapicommon.OpenAPIDefinition {
		result := apiextensionsopenapi.GetOpenAPIDefinitions(ref)
		for k, v := range defs(ref) {
			result[k] = v
		}

		// Add reverse-DNS alias entries alongside the original Go-path keys so that
		// BuildOpenAPIDefinitionsForResources can resolve canonical (reverse-DNS)
		// names that getResourceNamesForGroup computed via GetDefinitionName.
		aliases := make(map[string]openapicommon.OpenAPIDefinition)
		for k, v := range result {
			if strings.HasPrefix(k, "github.com/") {
				rdns := goImportPathToReverseDNS(k)
				if _, exists := result[rdns]; !exists {
					aliases[rdns] = v
				}
			}
		}
		for k, v := range aliases {
			result[k] = v
		}

		klog.V(4).Infof("mergedDefs: %d total definitions (%d reverse-DNS aliases added)", len(result), len(aliases))
		return result
	}

	options.RecommendedConfigFns = append(options.RecommendedConfigFns, func(config *server.RecommendedConfig) *server.RecommendedConfig {
		klog.Infof("apiserver-builder: applying OpenAPI config with reverse-DNS GetDefinitionName")

		config.OpenAPIConfig = server.DefaultOpenAPIConfig(mergedDefs, namer)
		config.OpenAPIConfig.Info.Title = name
		config.OpenAPIConfig.Info.Version = version
		config.OpenAPIConfig.GetDefinitionName = getDefinitionName

		config.OpenAPIV3Config = server.DefaultOpenAPIV3Config(mergedDefs, namer)
		config.OpenAPIV3Config.Info.Title = name
		config.OpenAPIV3Config.Info.Version = version
		config.OpenAPIV3Config.GetDefinitionName = getDefinitionName

		return config
	})

	return r
}

// WithoutEtcd removes etcd related settings from apiserver.
func (r *Server) WithoutEtcd() *Server {
	return r.WithOptionsFns(func(o *ServerOptions) *ServerOptions {
		o.RecommendedOptions.Etcd = nil
		return o
	})
}

func (r *Server) WithServerName(serverName string) *Server {
	r.ServerName = serverName
	return r
}

func (r *Server) WithEtcdPath(etcdPath string) *Server {
	r.EtcdPath = etcdPath
	return r
}