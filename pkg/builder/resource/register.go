package resource

import (
	"fmt"
	"net/url"
	"reflect"

	"github.com/henderiw/apiserver-builder/pkg/apiserver"
	"github.com/henderiw/apiserver-builder/pkg/builder/resource/resourcestrategy"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// AddToScheme returns a function to add the Objects to the scheme.
//
// AddToScheme will register the objects returned by New and NewList under the GroupVersion for each object.
// AddToScheme will also register the objects under the "__internal" group version for each object that
// returns true for IsStorageVersion.
// AddToScheme will register the defaulting function if it implements the Defaulter inteface.
func AddToScheme(objs ...Object) func(s *runtime.Scheme) error {
	return func(s *runtime.Scheme) error {
		for i := range objs {
			obj := objs[i]
			s.AddKnownTypes(obj.GetGroupVersionResource().GroupVersion(), obj.New(), obj.NewList())
			if obj.IsStorageVersion() {
				s.AddKnownTypes(schema.GroupVersion{
					Group:   obj.GetGroupVersionResource().Group,
					Version: runtime.APIVersionInternal,
				}, obj.New(), obj.NewList())
			} else {
				multiVersionObj, ok := obj.(MultiVersionObject)
				if !ok {
					return fmt.Errorf("resource should implement MultiVersionObject if it's not storage-version")
				}
				if err := multiVersionObj.RegisterConversions()(s); err != nil {
					return err
				}
			}
			if _, ok := obj.(resourcestrategy.Defaulter); ok {
				s.AddTypeDefaultingFunc(obj, func(o interface{}) {
					o.(resourcestrategy.Defaulter).Default()
				})
			}
			// register subresources
			if objWithStatus, ok := obj.(ObjectWithStatusSubResource); ok {
				if statusObj, ok := objWithStatus.GetStatus().(runtime.Object); ok {
					s.AddKnownTypes(obj.GetGroupVersionResource().GroupVersion(), statusObj)
				}
			}
			if _, ok := obj.(ObjectWithScaleSubResource); ok {
				if !s.Recognizes(autoscalingv1.SchemeGroupVersion.WithKind("Scale")) {
					if err := autoscalingv1.AddToScheme(s); err != nil {
						return err
					}
				}
			}

			// e.g. the scale subresources is adding an autoscaling scale kind to the type
			if arb, ok := obj.(ObjectWithArbitrarySubResource); ok {
				for _, sub := range arb.GetArbitrarySubResources() {
					subNew := sub.New()
					if reflect.TypeOf(subNew) != reflect.TypeOf(obj.New()) {
						s.AddKnownTypes(obj.GetGroupVersionResource().GroupVersion(), subNew)
					}
					// Register options type if the subresource uses GetterWithOptions
					if subWithOpts, ok := sub.(ArbitrarySubResourceWithOptions); ok {
						opts := subWithOpts.NewGetOptions()
						if opts != nil {
							s.AddKnownTypes(obj.GetGroupVersionResource().GroupVersion(), opts)
							apiserver.ParameterScheme.AddKnownTypes(obj.GetGroupVersionResource().GroupVersion(), opts)

							// Register url.Values → options conversion
							if converter, ok := sub.(ArbitrarySubResourceWithOptionsConverter); ok {
								if err := apiserver.ParameterScheme.AddConversionFunc(
									(*url.Values)(nil),
									opts,
									converter.ConvertFromURLValues(),
								); err != nil {
									return err
								}
								// Register version conversion for ParameterScheme
								if versionConverter, ok := sub.(ArbitrarySubResourceWithVersionConverter); ok {
									for _, convFn := range versionConverter.ParameterSchemeConversions() {
										if err := convFn(apiserver.ParameterScheme); err != nil {
											return err
										}
									}
								}
							}
						}
					}
				}
			}

		}
		return nil
	}
}
