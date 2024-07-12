package utils

import (
	"context"
	"fmt"
	"reflect"
	"strconv"

	"github.com/henderiw/apiserver-builder/pkg/builder/resource"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/conversion"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	genericapirequest "k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/generic"
	"k8s.io/apiserver/pkg/storage"
)

// MatchPackageRevision is the filter used by the generic etcd backend to watch events
// from etcd to clients of the apiserver only interested in specific labels/fields.
func Match(label labels.Selector, field fields.Selector) storage.SelectionPredicate {
	return storage.SelectionPredicate{
		Label:    label,
		Field:    field,
		GetAttrs: GetAttrs,
	}
}

// GetAttrs returns labels.Set, fields.Set, and error in case the given runtime.Object is not a ObjectMetaProvider
func GetAttrs(obj runtime.Object) (labels.Set, fields.Set, error) {
	provider, ok := obj.(resource.Object)
	if !ok {
		return nil, nil, fmt.Errorf("given object of type %T does not have metadata", obj)
	}
	om := provider.GetObjectMeta()
	return om.GetLabels(), SelectableFields(om), nil
}

// SelectableFields returns a field set that represents the object.
func SelectableFields(obj *metav1.ObjectMeta) fields.Set {
	return generic.ObjectMetaFieldsSet(obj, true)
}

// ParseFieldSelector parses client-provided fields.Selector into a storerFilter
func ParseFieldSelector(ctx context.Context, fieldSelector fields.Selector) (resource.Filter, error) {
	var filter *storerFilter
	// add the namespace to the list
	namespace, ok := genericapirequest.NamespaceFrom(ctx)
	if fieldSelector == nil {
		if ok {
			return &storerFilter{namespace: namespace}, nil
		}
		return filter, nil
	}

	requirements := fieldSelector.Requirements()
	for _, requirement := range requirements {
		filter = &storerFilter{}
		switch requirement.Operator {
		case selection.Equals, selection.DoesNotExist:
			if requirement.Value == "" {
				return filter, apierrors.NewBadRequest(fmt.Sprintf("unsupported fieldSelector value %q for field %q with operator %q", requirement.Value, requirement.Field, requirement.Operator))
			}
		default:
			return filter, apierrors.NewBadRequest(fmt.Sprintf("unsupported fieldSelector operator %q for field %q", requirement.Operator, requirement.Field))
		}

		switch requirement.Field {
		case "metadata.name":
			filter.name = requirement.Value
		case "metadata.namespace":
			filter.namespace = requirement.Value
		default:
			return filter, apierrors.NewBadRequest(fmt.Sprintf("unknown fieldSelector field %q", requirement.Field))
		}
	}
	// add namespace to the filter selector if specified
	if ok {
		if filter != nil {
			filter.namespace = namespace
		} else {
			filter = &storerFilter{namespace: namespace}
		}
	}

	return filter, nil
}

// Filter
type storerFilter struct {
	// Name filters by the name of the objects
	name string

	// Namespace filters by the namespace of the objects
	namespace string
}

func (r *storerFilter) Filter(ctx context.Context, obj runtime.Object) bool {
	f := false // this is the result of the previous filtering
	accessor, err := meta.Accessor(obj)
	if err != nil {
		return f
	}
	if r.name != "" {
		if accessor.GetName() == r.name {
			f = false
		} else {
			f = true
		}
	}
	if r.namespace != "" {
		if accessor.GetNamespace() == r.namespace {
			f = false
		} else {
			f = true
		}
	}
	return f
}

func UpdateResourceVersionAndGeneration(obj, old runtime.Object) error {
	accessorNew, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	accessorOld, err := meta.Accessor(old)
	if err != nil {
		return err
	}
	if err := updateResourceVersion(accessorNew, accessorOld); err != nil {
		return err
	}
	updateGeneration(accessorNew, accessorOld)
	return nil
}

func UpdateResourceVersion(obj, old runtime.Object) error {
	accessorNew, err := meta.Accessor(obj)
	if err != nil {
		return err
	}
	accessorOld, err := meta.Accessor(old)
	if err != nil {
		return err
	}
	if err := updateResourceVersion(accessorNew, accessorOld); err != nil {
		return err
	}
	updateGeneration(accessorNew, accessorOld)
	return nil
}

func updateResourceVersion(new, old metav1.Object) error {
	resourceVersion, err := strconv.Atoi(old.GetResourceVersion())
	if err != nil {
		return err
	}
	resourceVersion++
	new.SetResourceVersion(strconv.Itoa(resourceVersion))
	return nil
}

func updateGeneration(new, old metav1.Object) {
	generation := old.GetGeneration()
	generation++
	new.SetGeneration(generation)
}

func GetListPrt(listObj runtime.Object) (reflect.Value, error) {
	listPtr, err := meta.GetItemsPtr(listObj)
	if err != nil {
		return reflect.Value{}, err
	}
	v, err := conversion.EnforcePtr(listPtr)
	if err != nil || v.Kind() != reflect.Slice {
		return reflect.Value{}, fmt.Errorf("need ptr to slice: %v", err)
	}
	return v, nil
}

func AppendItem(v reflect.Value, obj runtime.Object) {
	v.Set(reflect.Append(v, reflect.ValueOf(obj).Elem()))
}
