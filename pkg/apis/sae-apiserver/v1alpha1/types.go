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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apiserver/pkg/registry/rest"
	"sigs.k8s.io/apiserver-runtime/pkg/builder/resource"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/kubevela/pkg/util/singleton"
)

var _ resource.ObjectWithArbitrarySubResource = &SAEAPIServer{}
var _ rest.Storage = &SAEAPIServer{}
var _ rest.Getter = &SAEAPIServer{}
var _ rest.Lister = &SAEAPIServer{}
var _ rest.Creater = &SAEAPIServer{}
var _ rest.Updater = &SAEAPIServer{}
var _ rest.Patcher = &SAEAPIServer{}
var _ rest.GracefulDeleter = &SAEAPIServer{}

// SAEAPIServer
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SAEAPIServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SAEAPIServerSpec `json:"spec,omitempty"`
}

func (in *SAEAPIServer) Destroy() {}

func (in *SAEAPIServer) GetObjectMeta() *metav1.ObjectMeta {
	return &in.ObjectMeta
}

func (in *SAEAPIServer) NamespaceScoped() bool {
	return false
}

func (in *SAEAPIServer) New() runtime.Object {
	return &SAEAPIServer{}
}

func (in *SAEAPIServer) NewList() runtime.Object {
	return &SAEAPIServerList{}
}

func (in *SAEAPIServer) GetGroupVersionResource() schema.GroupVersionResource {
	return GroupVersion.WithResource(SAEAPIServerResource)
}

func (in *SAEAPIServer) IsStorageVersion() bool {
	return true
}

func (in *SAEAPIServer) GetArbitrarySubResources() []resource.ArbitrarySubResource {
	return []resource.ArbitrarySubResource{&SAEAPIServerProxy{}}
}

// SAEAPIServerList
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SAEAPIServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SAEAPIServer `json:"items"`
}

type SAEAPIServerSpec struct {
	SAEAPIServerCredential `json:",inline"`
	Region                 string `json:"region,omitempty"`
}

type SAEAPIServerCredential struct {
	AccessKeyId     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
}

func (in *SAEAPIServer) Delete(ctx context.Context, name string, deleteValidation rest.ValidateObjectFunc, options *metav1.DeleteOptions) (runtime.Object, bool, error) {
	apiserver, err := in.Get(ctx, name, &metav1.GetOptions{})
	if err != nil {
		return nil, false, err
	}
	if err = singleton.StaticClient.Get().CoreV1().Secrets(storageNamespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return nil, false, err
	}
	return apiserver, true, nil
}

func (in *SAEAPIServer) Update(ctx context.Context, name string, objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc, updateValidation rest.ValidateObjectUpdateFunc, forceAllowCreate bool, options *metav1.UpdateOptions) (runtime.Object, bool, error) {
	apiserver, err := in.Get(ctx, name, &metav1.GetOptions{})
	if err != nil {
		return nil, false, err
	}
	if apiserver, err = objInfo.UpdatedObject(ctx, apiserver); err != nil {
		return nil, false, err
	}
	secret := convertSAEAPIServerToSecret(apiserver.(*SAEAPIServer))
	if err = singleton.KubeClient.Get().Update(ctx, secret); err != nil {
		return nil, false, err
	}
	return apiserver, true, nil
}

func (in *SAEAPIServer) Create(ctx context.Context, obj runtime.Object, createValidation rest.ValidateObjectFunc, options *metav1.CreateOptions) (runtime.Object, error) {
	apiserver := obj.(*SAEAPIServer)
	secret := convertSAEAPIServerToSecret(apiserver)
	var err error
	if secret, err = singleton.StaticClient.Get().CoreV1().Secrets(storageNamespace).Create(ctx, secret, metav1.CreateOptions{}); err != nil {
		return nil, err
	}
	if apiserver, err = convertSecretToSAEAPIServer(secret); err != nil {
		return nil, err
	}
	return apiserver, err
}

func (in *SAEAPIServer) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	secrets := &corev1.SecretList{}
	if err := singleton.KubeClient.Get().List(ctx, secrets, client.InNamespace(storageNamespace), client.MatchingLabels{LabelSAEAPIServer: LabelKeySAEAPIServer}); err != nil {
		return nil, err
	}
	apiservers := &SAEAPIServerList{}
	for _, secret := range secrets.Items {
		apiserver, err := convertSecretToSAEAPIServer(secret.DeepCopy())
		if err != nil {
			return nil, err
		}
		apiservers.Items = append(apiservers.Items, *apiserver)
	}
	return apiservers, nil
}

func (in *SAEAPIServer) Get(ctx context.Context, name string, options *metav1.GetOptions) (runtime.Object, error) {
	secret := &corev1.Secret{}
	if err := singleton.KubeClient.Get().Get(ctx, types.NamespacedName{Namespace: storageNamespace, Name: name}, secret); err != nil {
		return nil, err
	}
	return convertSecretToSAEAPIServer(secret)
}
