/*
Copyright 2020 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resource

import (
	"context"

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/registry/rest"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// Object must be implemented by all resources served by the apiserver.
type Object interface {
	// Object allows the apiserver libraries to operate on the Object
	runtime.Object

	// GetObjectMeta returns the object meta reference.
	GetObjectMeta() *metav1.ObjectMeta

	// Scoper is used to qualify the resource as either namespace scoped or non-namespace scoped.
	rest.Scoper

	// New returns a new instance of the resource -- e.g. &v1.Deployment{}
	New() runtime.Object

	// NewList return a new list instance of the resource -- e.g. &v1.DeploymentList{}
	NewList() runtime.Object

	// GetGroupVersionResource returns the GroupVersionResource for this resource.  The resource should
	// be the all lowercase and pluralized kind.s
	GetGroupVersionResource() schema.GroupVersionResource

	// IsStorageVersion returns true if the object is also the internal version -- i.e. is the type defined
	// for the API group an alias to this object.
	// If false, the resource is expected to implement MultiVersionObject interface.
	IsStorageVersion() bool
}

type InternalObject interface {
	Object

	// GetSingularName return the singular name of the resource
	GetSingularName() string

	// GetShortNames retruns the short names for the resource
	GetShortNames() []string

	// GetCategories returns the categories of the resource
	GetCategories() []string

	// NamespaceScoped returns if the resource is namespaced or not
	NamespaceScoped() bool

	// TableConvertor return the table convertor interface
	TableConvertor() func(gr schema.GroupResource) rest.TableConvertor

	// FieldLabelConversion returns the field conversion function
	FieldLabelConversion() runtime.FieldLabelConversionFunc

	// FieldSelector returns a function that is used to filter resources based on field selectors
	FieldSelector() func(ctx context.Context, fieldSelector fields.Selector) (Filter, error)

	// PrepareForCreate prepares the resource for creation.
	// e.g. sets status empty status
	PrepareForCreate(ctx context.Context)

	// ValidateCreate return field errors on specific validation of the resource upon create
	ValidateCreate(ctx context.Context) field.ErrorList

	// PrepareForUpdate prepares the resource for update.
	// e.g. sets status empty status
	PrepareForUpdate(ctx context.Context, old runtime.Object)

	// ValidateUpdate return field errors on specific validation of the resource upon update
	ValidateUpdate(ctx context.Context, old runtime.Object) field.ErrorList

	// Is Equal compares the specification of the various objects
	IsEqual(ctx context.Context, obj, old runtime.Object) bool
}

// ObjectList must be implemented by all resources' list object.
type ObjectList interface {
	// Object allows the apiserver libraries to operate on the Object
	runtime.Object

	// GetListMeta returns the list meta reference.
	GetListMeta() *metav1.ListMeta
}

// MultiVersionObject should be implemented if the resource is not storage version and has multiple versions serving
// at the server.
type MultiVersionObject interface {
	RegisterConversions() func(s *runtime.Scheme) error
}

// ObjectWithStatusSubResource defines an interface for getting and setting the status sub-resource for a resource.
type ObjectWithStatusSubResource interface {
	Object
	GetStatus() (statusSubResource StatusSubResource)
}

// ObjectWithScaleSubResource defines an interface for getting and setting the scale sub-resource for a resource.
type ObjectWithScaleSubResource interface {
	Object
	SetScale(scaleSubResource *autoscalingv1.Scale)
	GetScale() (scaleSubResource *autoscalingv1.Scale)
}

// ObjectWithArbitrarySubResource defines an interface for plumbing arbitrary sub-resources for a resource.
type ObjectWithArbitrarySubResource interface {
	Object
	GetArbitrarySubResources() []ArbitrarySubResource
}

type Filter interface {
	Filter(ctx context.Context, obj runtime.Object) bool
}
