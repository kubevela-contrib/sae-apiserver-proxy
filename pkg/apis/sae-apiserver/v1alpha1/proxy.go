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
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	registryrest "k8s.io/apiserver/pkg/registry/rest"
	"k8s.io/apiserver/pkg/server"
	"k8s.io/utils/strings/slices"
	"sigs.k8s.io/apiserver-runtime/pkg/builder/resource"
	"sigs.k8s.io/apiserver-runtime/pkg/builder/resource/resourcerest"
	contextutil "sigs.k8s.io/apiserver-runtime/pkg/util/context"
)

var _ resource.SubResource = &SAEAPIServerProxy{}
var _ registryrest.Storage = &SAEAPIServerProxy{}
var _ resourcerest.Connecter = &SAEAPIServerProxy{}

var proxyMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

type SAEAPIServerProxy struct{}

// SAEAPIServerProxyOptions
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SAEAPIServerProxyOptions struct {
	metav1.TypeMeta

	Path string `json:"path"`
}

var _ resource.QueryParameterObject = &SAEAPIServerProxyOptions{}

func (in *SAEAPIServerProxyOptions) ConvertFromUrlValues(values *url.Values) error {
	in.Path = values.Get("path")
	return nil
}

func (in *SAEAPIServerProxy) NewConnectOptions() (runtime.Object, bool, string) {
	return &SAEAPIServerProxyOptions{}, true, "path"
}

func (in *SAEAPIServerProxy) ConnectMethods() []string {
	return proxyMethods
}

func (in *SAEAPIServerProxy) New() runtime.Object {
	return &SAEAPIServerProxyOptions{}
}

func (in *SAEAPIServerProxy) Destroy() {}

func (in *SAEAPIServerProxy) SubResourceName() string {
	return "proxy"
}

func (in *SAEAPIServerProxy) Connect(ctx context.Context, id string, options runtime.Object, r registryrest.Responder) (http.Handler, error) {
	opts, ok := options.(*SAEAPIServerProxyOptions)
	if !ok {
		return nil, fmt.Errorf("invalid options object: %#v", options)
	}

	parentStorage, ok := contextutil.GetParentStorageGetter(ctx)
	if !ok {
		return nil, fmt.Errorf("no parent storage found")
	}
	parentObj, err := parentStorage.Get(ctx, id, &metav1.GetOptions{})

	if err != nil {
		return nil, fmt.Errorf("no such cluster %v", id)
	}
	apiserver := parentObj.(*SAEAPIServer)

	cli, err := sdk.NewClientWithAccessKey(apiserver.Spec.Region, apiserver.Spec.AccessKeyId, apiserver.Spec.AccessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("cannot create alibaba-cloud client with ak/sk: %w", err)
	}

	return &proxyHandler{
		apiserver: apiserver,
		path:      opts.Path,
		responder: r,
		cli:       cli,
	}, nil
}

const (
	saeVersion      = "2019-05-06"
	saeAPIName      = "VirtualServerProxy"
	saeProduct      = "sae"
	saeServiceCode  = "serverless"
	saeEndpointType = "openAPI"
)

type proxyHandler struct {
	apiserver *SAEAPIServer
	path      string
	responder registryrest.Responder
	cli       *sdk.Client
}

func (in *proxyHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	response, err := in.RoundTrip(request)
	if err == nil {
		var body []byte
		body, err = io.ReadAll(response.Body)
		if err == nil {
			for key, values := range response.Header {
				for _, val := range values {
					writer.Header().Add(key, val)
				}
			}
			writer.WriteHeader(response.StatusCode)
			_, err = writer.Write(body)
		}
	}
	if err != nil {
		in.responder.Error(err)
	}
}

func (in *proxyHandler) RoundTrip(httpReq *http.Request) (*http.Response, error) {
	req := requests.NewCommonRequest()
	req.Scheme = requests.HTTPS
	req.PathPattern = "/pop/v1/apiserver/proxy"
	reqPath := strings.TrimPrefix(in.path, path.Join("/apis", Group, Version, SAEAPIServerResource, in.apiserver.Name, "proxy"))
	if query := unescapeQueryValues(httpReq.URL.Query()); len(query) > 0 {
		reqPath += "?" + query.Encode()
	}
	body := &input{
		Path:        reqPath,
		Method:      httpReq.Method,
		ContentType: requests.Json,
		Header:      httpReq.Header,
	}
	if httpReq.Body != nil {
		data, _ := io.ReadAll(httpReq.Body)
		body.Content = string(data)
	}
	req.SetContent(body.json())
	req.Version = saeVersion
	req.ApiName = saeAPIName
	req.Product = saeProduct
	req.ServiceCode = saeServiceCode
	req.EndpointType = saeEndpointType
	req.Method = requests.POST
	req.SetContentType(requests.Json)
	response, err := in.cli.ProcessCommonRequest(req)
	if err != nil {
		return nil, err
	}
	httpResponse := &http.Response{}
	httpResponse.ProtoMajor = httpReq.ProtoMajor
	httpResponse.Proto = httpReq.Proto
	httpResponse.ProtoMinor = httpReq.ProtoMinor
	httpResponse.Request = httpReq
	out := &output{}
	if err = json.Unmarshal(response.GetHttpContentBytes(), out); err != nil {
		return nil, err
	}
	httpResponse.Header = out.Header
	httpResponse.StatusCode = out.Code
	data, err := base64.StdEncoding.DecodeString(out.Body)
	if err != nil {
		return nil, err
	}
	httpResponse.Body = io.NopCloser(bytes.NewReader(data))
	httpResponse.ContentLength = int64(len(data))
	return httpResponse, nil
}

type input struct {
	Path        string              `json:"path"`
	Method      string              `json:"method"`
	ContentType string              `json:"contentType"`
	Content     string              `json:"content"`
	Header      map[string][]string `json:"header,omitempty"`
}

type output struct {
	RequestId string              `json:"requestId"`
	Code      int                 `json:"code"`
	Error     string              `json:"error,omitempty"`
	Body      string              `json:"body,omitempty"`
	Header    map[string][]string `json:"header,omitempty"`
}

func (in *input) json() []byte {
	bt, _ := json.Marshal(in)
	return bt
}

func NewProxyRequestEscaper(delegate http.Handler) http.Handler {
	return &proxyRequestEscaper{delegate: delegate}
}

type proxyRequestEscaper struct {
	delegate http.Handler
}

var (
	proxyPathPattern = regexp.MustCompile(strings.Join([]string{
		server.APIGroupPrefix,
		Group,
		Version,
		SAEAPIServerResource,
		"[a-z0-9]([-a-z0-9]*[a-z0-9])?",
		"proxy"}, "/"))
	proxyQueryKeysToEscape = []string{"dryRun"}
	proxyEscaperPrefix     = "__"
)

func (in *proxyRequestEscaper) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if proxyPathPattern.MatchString(req.URL.Path) {
		newReq := req.Clone(req.Context())
		q := req.URL.Query()
		for _, k := range proxyQueryKeysToEscape {
			if q.Has(k) {
				q.Set(proxyEscaperPrefix+k, q.Get(k))
				q.Del(k)
			}
		}
		newReq.URL.RawQuery = q.Encode()
		req = newReq
	}
	in.delegate.ServeHTTP(w, req)
}

func unescapeQueryValues(values url.Values) url.Values {
	unescaped := url.Values{}
	for k, vs := range values {
		if strings.HasPrefix(k, proxyEscaperPrefix) &&
			slices.Contains(proxyQueryKeysToEscape,
				strings.TrimPrefix(k, proxyEscaperPrefix)) {
			k = strings.TrimPrefix(k, proxyEscaperPrefix)
		}
		unescaped[k] = vs
	}
	return unescaped
}
