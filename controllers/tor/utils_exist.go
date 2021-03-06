/*
Copyright 2021.

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

package tor

import (
	"reflect"

	corev1 "k8s.io/api/core/v1"
)

// func itemExists(slice interface{}, item interface{}) bool {
// 	s := reflect.ValueOf(slice)

// 	if s.Kind() != reflect.Slice {
// 		panic("Invalid data-type")
// 	}

// 	for i := 0; i < s.Len(); i++ {
// 		if s.Index(i).Interface() == item {
// 			return true
// 		}
// 	}

// 	return false
// }

func portExists(slice []corev1.ServicePort, item corev1.ServicePort) bool {
	s := reflect.ValueOf(slice)

	if s.Kind() != reflect.Slice {
		panic("Invalid data-type")
	}

	for _, p := range slice {
		if p.Protocol == item.Protocol {
			if p.Port == item.Port {
				return true
			}
			if p.Name != "" && p.Name == item.Name {
				return true
			}
		}
	}

	return false
}
