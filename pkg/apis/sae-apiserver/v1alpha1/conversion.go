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
	"fmt"

	"github.com/kubevela/pkg/util/k8s"
	"github.com/kubevela/pkg/util/singleton"
	"github.com/oam-dev/cluster-gateway/pkg/apis/cluster/v1alpha1"
	"github.com/oam-dev/cluster-gateway/pkg/common"
	corev1 "k8s.io/api/core/v1"
)

const (
	IdentAccessKeyId          = "accessKeyId"
	IdentAccessKeySecret      = "accessKeySecret"
	LabelSAEAPIServer         = "sae.alibaba-cloud.oam.dev/apiserver"
	LabelKeySAEAPIServer      = "true"
	LabelSAEAPIServerRegion   = "sae.alibaba-cloud.oam.dev/apiserver-region"
	DefaultSAEAPIServerRegion = "cn-hangzhou"
)

func convertSecretToSAEAPIServer(secret *corev1.Secret) (*SAEAPIServer, error) {
	apiserver := &SAEAPIServer{}
	apiserver.ObjectMeta = secret.ObjectMeta
	accessKeyId, f1 := secret.Data[IdentAccessKeyId]
	accessKeySecret, f2 := secret.Data[IdentAccessKeySecret]
	if !f1 || !f2 {
		return nil, fmt.Errorf("accessKey not found in secret %s/%s", storageNamespace, secret.Name)
	}
	if isAPIServer := k8s.GetLabel(secret, LabelSAEAPIServer); isAPIServer != LabelKeySAEAPIServer {
		return nil, fmt.Errorf("secret %s/%s is not a SAEAPIServer secret", storageNamespace, secret.Name)
	}
	if apiserver.Spec.Region = k8s.GetLabel(secret, LabelSAEAPIServerRegion); apiserver.Spec.Region == "" {
		apiserver.Spec.Region = DefaultSAEAPIServerRegion
	}
	apiserver.Spec.AccessKeyId = string(accessKeyId)
	apiserver.Spec.AccessKeySecret = string(accessKeySecret)
	return apiserver, nil
}

func convertSAEAPIServerToSecret(apiserver *SAEAPIServer) *corev1.Secret {
	secret := &corev1.Secret{Data: map[string][]byte{}}
	secret.ObjectMeta = apiserver.ObjectMeta
	secret.SetNamespace(apiserver.Name)
	secret.SetNamespace(storageNamespace)
	region := apiserver.Spec.Region
	if region == "" {
		region = DefaultSAEAPIServerRegion
	}
	_ = k8s.AddLabel(secret, LabelSAEAPIServerRegion, region)
	_ = k8s.AddLabel(secret, LabelSAEAPIServer, LabelKeySAEAPIServer)
	secret.Data[IdentAccessKeyId] = []byte(apiserver.Spec.AccessKeyId)
	secret.Data[IdentAccessKeySecret] = []byte(apiserver.Spec.AccessKeySecret)
	attachClusterGatewayMetadata(secret)
	return secret
}

func attachClusterGatewayMetadata(secret *corev1.Secret) {
	cfg := singleton.KubeConfig.Get()
	if cfg.TLSClientConfig.CertData != nil && cfg.TLSClientConfig.KeyData != nil {
		secret.Data["tls.crt"] = cfg.TLSClientConfig.CertData
		secret.Data["tls.key"] = cfg.TLSClientConfig.KeyData
		_ = k8s.AddLabel(secret, common.LabelKeyClusterCredentialType, string(v1alpha1.CredentialTypeX509Certificate))
	} else if len(cfg.BearerToken) > 0 {
		secret.Data["token"] = []byte(cfg.BearerToken)
		_ = k8s.AddLabel(secret, common.LabelKeyClusterCredentialType, string(v1alpha1.CredentialTypeServiceAccountToken))
	}
	secret.Data["endpoint"] = []byte(fmt.Sprintf("%s/apis/%s/%s/%s/%s/proxy/", serverAddress, Group, Version, SAEAPIServerResource, secret.Name))
}
