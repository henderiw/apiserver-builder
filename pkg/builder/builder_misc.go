package builder

import (
	"strings"

	"github.com/henderiw/apiserver-builder/pkg/apiserver"
	"github.com/henderiw/apiserver-builder/pkg/cmd/apiserverbuilder/options"
	apiextensionsopenapi "k8s.io/apiextensions-apiserver/pkg/generated/openapi"
	openapinamer "k8s.io/apiserver/pkg/endpoints/openapi"
	"k8s.io/apiserver/pkg/server"
	scheme "k8s.io/client-go/kubernetes/scheme"
	openapicommon "k8s.io/kube-openapi/pkg/common"
	spec "k8s.io/kube-openapi/pkg/validation/spec"
)

// goImportPathToReverseDNS converts a Go import path to a reverse-DNS OpenAPI
// component name using the same convention as core Kubernetes types.
//
//	github.com/sdcio/config-server/apis/config/v1alpha1.ConfigStatus
//	→ com.github.sdcio.config-server.apis.config.v1alpha1.ConfigStatus
//
//	k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta
//	→ io.k8s.apimachinery.pkg.apis.meta.v1.ObjectMeta
//
// The resulting name contains neither "/" nor "~", so kube-openapi's
// EscapeJsonPointer is a no-op and $ref values are always stable.
func goImportPathToReverseDNS(goPath string) string {
	slash := strings.Index(goPath, "/")
	if slash == -1 {
		// No host segment — return unchanged (e.g. stdlib or simple names).
		return goPath
	}
	host := goPath[:slash]
	rest := strings.ReplaceAll(goPath[slash+1:], "/", ".")

	// Reverse the host domain labels: "github.com" → "com.github"
	parts := strings.Split(host, ".")
	for i, j := 0, len(parts)-1; i < j; i, j = i+1, j-1 {
		parts[i], parts[j] = parts[j], parts[i]
	}
	return strings.Join(parts, ".") + "." + rest
}

// WithOpenAPIDefinitions registers OpenAPI definitions for the API server.
//
// # Why a custom GetDefinitionName is needed
//
// openapinamer.NewDefinitionNamer produces clean reverse-DNS names (e.g.
// com.github.sdcio…v1alpha1.Config) only for types registered in the scheme
// with a GVK.  For embedded/sub-types that are NOT GVK roots — ConfigStatus,
// TargetStatus, DeviationSpec, etc. — it falls back to:
//
//	strings.ReplaceAll(name, "/", "~1")
//
// producing names like "github.com~1sdcio~1…ConfigStatus".
//
// kube-openapi then calls EscapeJsonPointer() on that name when building
// $ref strings, which escapes the "~" to "~0", yielding "~01" — a
// double-encoding.  Both kubectl's JSON-Pointer resolver and the
// structured-merge-diff TypeConverter used by SSA cannot consistently
// reconcile "~01"-encoded $ref values against "~1"-keyed components,
// causing one of two failure modes:
//
//   - kubectl client-side validation: $ref target not found → validation error
//   - SSA TypeConverter: "no type found matching: github.com~1…ConfigStatus"
//
// Fix: wrap the namer so that any fallback result containing "/" or "~"
// is replaced by a proper goImportPathToReverseDNS name.  Reverse-DNS
// names need no JSON Pointer escaping, so $ref and component-key always
// match exactly.
func (r *Server) WithOpenAPIDefinitions(
	name, version string,
	defs openapicommon.GetOpenAPIDefinitions) *Server {

	namer := openapinamer.NewDefinitionNamer(apiserver.Scheme, scheme.Scheme)

	// getDefinitionName delegates to the namer for scheme-registered GVK types
	// (preserving their x-kubernetes-group-version-kind extensions), and applies
	// proper reverse-DNS for everything else.
	getDefinitionName := func(n string) (string, spec.Extensions) {
		resolved, ext := namer.GetDefinitionName(n)
		// The namer's fallback produces "~1"-encoded names; detect and replace.
		if strings.ContainsAny(resolved, "/~") {
			return goImportPathToReverseDNS(n), nil
		}
		return resolved, ext
	}

	mergedDefs := func(ref openapicommon.ReferenceCallback) map[string]openapicommon.OpenAPIDefinition {
		result := apiextensionsopenapi.GetOpenAPIDefinitions(ref)
		for k, v := range defs(ref) {
			result[k] = v
		}
		return result
	}

	options.RecommendedConfigFns = append(options.RecommendedConfigFns, func(config *server.RecommendedConfig) *server.RecommendedConfig {
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