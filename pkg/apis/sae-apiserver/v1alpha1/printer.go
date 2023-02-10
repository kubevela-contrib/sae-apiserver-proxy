/*
Copyright 2022 The KubeVela Authors.

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

package v1alpha1

import (
	"context"
	"fmt"

	"github.com/kubevela/pkg/util/slices"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func (in *SAEAPIServer) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	switch o := object.(type) {
	case tableConverter:
		return o.ToTable(), nil
	default:
		return nil, fmt.Errorf("unsupported type for table conversion: %T", object)
	}
}

var (
	definitions = []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: "the name of the SAEAPIServer"},
		{Name: "Region", Type: "string", Description: "the region of the SAEAPIServer"},
		{Name: "AK", Type: "string", Description: "the accessKeyId of the SAEAPIServer"},
	}
)

func (in *SAEAPIServer) row() *metav1.TableRow {
	return &metav1.TableRow{
		Object: runtime.RawExtension{Object: in},
		Cells: []interface{}{
			in.Name,
			in.Spec.Region,
			in.Spec.AccessKeyId,
		},
	}
}

type tableConverter interface {
	ToTable() *metav1.Table
}

func (in *SAEAPIServer) ToTable() *metav1.Table {
	return &metav1.Table{
		ColumnDefinitions: definitions,
		Rows:              []metav1.TableRow{*in.row()},
	}
}

func (in *SAEAPIServerList) ToTable() *metav1.Table {
	return &metav1.Table{
		ColumnDefinitions: definitions,
		Rows: slices.Map(in.Items, func(item SAEAPIServer) metav1.TableRow {
			return *item.row()
		}),
	}
}
