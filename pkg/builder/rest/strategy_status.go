package rest

/*

import (
	"context"

	"github.com/henderiw/apiserver-builder/pkg/builder/resource"
	"github.com/henderiw/apiserver-builder/pkg/builder/resource/resourcestrategy"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/storage"
	"k8s.io/apiserver/pkg/storage/names"
)

// StatusStrategy defines functions that are invoked prior to storing a Kubernetes resource.
type StatusStrategy interface {
}

var _ StatusStrategy = DefaultStatusStrategy{}

// DefaultStatusStrategy implements StatusStrategy.
type DefaultStatusStrategy struct {
	Object runtime.Object
	names.NameGenerator
	runtime.ObjectTyper
}

// GenerateName generates a new name for a resource without one.
func (d DefaultStatusStrategy) GenerateName(base string) string {
	if d.Object == nil {
		return names.SimpleNameGenerator.GenerateName(base)
	}
	if n, ok := d.Object.(names.NameGenerator); ok {
		return n.GenerateName(base)
	}
	return names.SimpleNameGenerator.GenerateName(base)
}

// NamespaceScoped is used to register the resource as namespaced or non-namespaced.
func (d DefaultStatusStrategy) NamespaceScoped() bool {
	if d.Object == nil {
		return true
	}
	if n, ok := d.Object.(rest.Scoper); ok {
		return n.NamespaceScoped()
	}
	return true
}

// PrepareForCreate calls the PrepareForCreate function on obj if supported, otherwise does nothing.
func (DefaultStatusStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
	if v, ok := obj.(resourcestrategy.PrepareForCreater); ok {
		v.PrepareForCreate(ctx)
	}
}

// PrepareForUpdate calls the PrepareForUpdate function on obj if supported, otherwise does nothing.
func (DefaultStatusStrategy) PrepareForUpdate(ctx context.Context, obj, old runtime.Object) {
	if v, ok := obj.(resource.ObjectWithStatusSubResource); ok {
		// don't modify the status
		old.(resource.ObjectWithStatusSubResource).GetStatus().CopyTo(v)
	}
	if v, ok := obj.(resourcestrategy.PrepareForUpdater); ok {
		v.PrepareForUpdate(ctx, old)
	}
}

// Validate calls the Validate function on obj if supported, otherwise does nothing.
func (DefaultStatusStrategy) Validate(ctx context.Context, obj runtime.Object) field.ErrorList {
	if v, ok := obj.(resourcestrategy.Validater); ok {
		return v.Validate(ctx)
	}
	return field.ErrorList{}
}

// AllowCreateOnUpdate is used by the Store
func (d DefaultStatusStrategy) AllowCreateOnUpdate() bool {
	if d.Object == nil {
		return false
	}
	if n, ok := d.Object.(resourcestrategy.AllowCreateOnUpdater); ok {
		return n.AllowCreateOnUpdate()
	}
	return false
}

// AllowUnconditionalUpdate is used by the Store
func (d DefaultStatusStrategy) AllowUnconditionalUpdate() bool {
	if d.Object == nil {
		return false
	}
	if n, ok := d.Object.(resourcestrategy.AllowUnconditionalUpdater); ok {
		return n.AllowUnconditionalUpdate()
	}
	return false
}

// Canonicalize calls the Canonicalize function on obj if supported, otherwise does nothing.
func (DefaultStatusStrategy) Canonicalize(obj runtime.Object) {
	if c, ok := obj.(resourcestrategy.Canonicalizer); ok {
		c.Canonicalize()
	}
}

// ValidateUpdate calls the ValidateUpdate function on obj if supported, otherwise does nothing.
func (DefaultStatusStrategy) ValidateUpdate(ctx context.Context, obj, old runtime.Object) field.ErrorList {
	if v, ok := obj.(resourcestrategy.ValidateUpdater); ok {
		return v.ValidateUpdate(ctx, old)
	}
	return field.ErrorList{}
}

// Match is the filter used by the generic etcd backend to watch events
// from etcd to clients of the apiserver only interested in specific labels/fields.
func (DefaultStatusStrategy) Match(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// ConvertToTable is used for printing the resource from kubectl get
func (d DefaultStatusStrategy) ConvertToTable(
	ctx context.Context, obj runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	if c, ok := obj.(resourcestrategy.TableConverter); ok {
		return c.ConvertToTable(ctx, tableOptions)
	}
	return d.TableConvertor.ConvertToTable(ctx, obj, tableOptions)
}

// WarningsOnCreate sends warning header on create
func (d DefaultStatusStrategy) WarningsOnCreate(ctx context.Context, obj runtime.Object) []string {
	return nil
}

// WarningsOnUpdate sends warning header on update
func (d DefaultStatusStrategy) WarningsOnUpdate(ctx context.Context, obj, old runtime.Object) []string {
	return nil
}
*/