package resource

import (
	"fmt"
	"reflect"

	"github.com/henderiw/apiserver-builder/pkg/builder/resource/resourcestrategy"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	"k8s.io/apimachinery/pkg/conversion"
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
				fmt.Println("AddToScheme storageVersion", obj.GetGroupVersionResource().GroupVersion().String(), obj, reflect.TypeOf(obj.New()).Name())
			} else {
				multiVersionObj, ok := obj.(MultiVersionObject)
				if !ok {
					return fmt.Errorf("resource should implement MultiVersionObject if it's not storage-version")
				}
				// registering conversion functions to scheme instance
				storageVersionObj := multiVersionObj.NewStorageVersionObject()
				if err := s.AddConversionFunc(obj, storageVersionObj, func(from, to interface{}, _ conversion.Scope) error {
					return from.(MultiVersionObject).ConvertToStorageVersion(to.(runtime.Object))
				}); err != nil {
					return err
				}
				if err := s.AddConversionFunc(storageVersionObj, obj, func(from, to interface{}, _ conversion.Scope) error {
					return to.(MultiVersionObject).ConvertFromStorageVersion(from.(runtime.Object))
				}); err != nil {
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
					// WIMCHECK IS THIS RELECTING /status
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

			// WIM: we assume right now that an arbitrary subresource is using the parent object
			// e.g. the scale subresources is adding an autoscaling scale kind to the type
			/*
				if sgs, ok := obj.(ObjectWithArbitrarySubResource); ok {
					for _, sub := range sgs.GetArbitrarySubResources() {
						sub := sub
						parentGVR := obj.GetGroupVersionResource()
						subResourceGVR := parentGVR.GroupVersion().WithResource(parentGVR.Resource + "/" + sub.SubResourceName())
						if reflect.TypeOf(sub.New()) != reflect.TypeOf(obj) { // subResource.New() may return the parent resource at some time
							fmt.Println("AddToScheme: arbitrary resource")
							s.AddKnownTypes(subResourceGVR.GroupVersion(), sub.New())
						}
					}
				}
			*/
		}
		return nil
	}
}
