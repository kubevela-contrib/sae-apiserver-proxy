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

package main

import (
	"github.com/kubevela/pkg/util/log"
	"k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/apiserver-runtime/pkg/builder"

	apiserveroptions "github.com/kubevela/pkg/util/apiserver/options"

	"github.com/kubevela-contrib/sae-apiserver-proxy/pkg/apis/sae-apiserver/v1alpha1"
)

func main() {
	cmd, err := builder.APIServer.
		WithLocalDebugExtension().
		ExposeLoopbackMasterClientConfig().
		ExposeLoopbackAuthorizer().
		WithResource(&v1alpha1.SAEAPIServer{}).
		WithoutEtcd().
		WithServerFns(func(server *builder.GenericAPIServer) *builder.GenericAPIServer {
			server.Handler.FullHandlerChain = v1alpha1.NewProxyRequestEscaper(server.Handler.FullHandlerChain)
			return server
		}).
		Build()
	runtime.Must(err)
	log.AddLogFlags(cmd)
	v1alpha1.AddFlags(cmd.Flags())
	apiserveroptions.AddServerRunFlags(cmd.Flags())
	runtime.Must(cmd.Execute())
}
