package builder

import (
	"github.com/henderiw/apiserver-builder/pkg/cmd/apiserverbuilder"
	"github.com/spf13/cobra"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/kube-openapi/pkg/common"
	//builderrest "github.com/henderiw/apiserver-builder/pkg/builder/rest"
)

// GenericAPIServer is an alias for pkgserver.GenericAPIServer
type GenericAPIServer = server.GenericAPIServer

// ServerOptions is an alias for apiserverbuilder.ServerOptions
type ServerOptions = apiserverbuilder.ServerOptions

// OpenAPIDefinition is an alias for common.OpenAPIDefinition
type OpenAPIDefinition = common.OpenAPIDefinition

// Storage is an alias for rest.Storage.  Storage implements the interfaces defined in the rest package
// to expose new REST endpoints for a Kubernetes resource.
type Storage = rest.Storage

// Command is an alias for cobra.Command and is used to start the apiserver.
type Command = cobra.Command

// DefaultStrategy is a default strategy that may be embedded into other strategies
//type DefaultStrategy = builderrest.DefaultStrategy
