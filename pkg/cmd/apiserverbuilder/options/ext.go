package options

import (
	"github.com/spf13/pflag"
	"k8s.io/apiserver/pkg/server"
)

var (
	EtcdPath             string
	RecommendedConfigFns []func(*server.RecommendedConfig) *server.RecommendedConfig
	ServerOptionsFns     []func(server *ServerOptions) *ServerOptions
	FlagsFns             []func(fs *pflag.FlagSet) *pflag.FlagSet
)

func ApplyServerOptionsFns(in *ServerOptions) *ServerOptions {
	for i := range ServerOptionsFns {
		in = ServerOptionsFns[i](in)
	}
	return in
}

func ApplyRecommendedConfigFns(in *server.RecommendedConfig) *server.RecommendedConfig {
	for i := range RecommendedConfigFns {
		in = RecommendedConfigFns[i](in)
	}
	return in
}

func ApplyFlagsFns(fs *pflag.FlagSet) *pflag.FlagSet {
	for i := range FlagsFns {
		fs = FlagsFns[i](fs)
	}
	return fs
}
