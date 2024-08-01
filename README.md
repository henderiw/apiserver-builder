# README

Example usage:

1. define a custom apiserver name if you dont want to use the default one
2. identify your openapi defintions you generated
3. Add your resources with the respective storage provider
4. 

```go
if err := builder.APIServer.
    WithServerName("your-custom-apiserver").
    WithOpenAPIDefinitions("Config", "v1alpha1", <generated openapi-definition>).
    WithResourceAndHandler(<api-resource>, <storageprovider>).
    WithoutEtcd().
    Execute(ctx); err != nil {
    log.Info("cannot start customer-apiserver")
}
```

e.g.

```go
if err := builder.APIServer.
    WithServerName("config-server").
    WithOpenAPIDefinitions("Config", "v1alpha1", configopenapi.GetOpenAPIDefinitions).
    WithResourceAndHandler(&config.Config{}, configStorageProvider).
    WithResourceAndHandler(&configv1alpha1.Config{}, configStorageProvider).
    WithResourceAndHandler(&config.ConfigSet{}, configSetStorageProvider).
    WithResourceAndHandler(&configv1alpha1.ConfigSet{}, configSetStorageProvider).
    WithResourceAndHandler(&config.UnManagedConfig{}, unmanagedConfigStorageProvider).
    WithResourceAndHandler(&configv1alpha1.UnManagedConfig{}, unmanagedConfigStorageProvider).
    WithResourceAndHandler(&config.RunningConfig{}, runningConfigStorageProvider).
    WithResourceAndHandler(&configv1alpha1.RunningConfig{}, runningConfigStorageProvider).
    WithoutEtcd().
    Execute(ctx); err != nil {
    log.Info("cannot start config-server")
}
```