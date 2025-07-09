//go:build integ
// +build integ

// Copyright Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ambient

import (
	// "context"
	// "fmt"
	"testing"
	// "time"
	//
	// corev1 "k8s.io/api/core/v1"
	// metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	// "sigs.k8s.io/yaml"

	"istio.io/istio/pkg/test/framework"
	// "istio.io/istio/pkg/test/framework/components/cluster"
	// "istio.io/istio/pkg/test/framework/components/echo"
	// "istio.io/istio/pkg/test/echo/common/scheme"
	"istio.io/istio/pkg/test/framework/components/echo/check"
	// "istio.io/istio/pkg/test/util/retry"
	// "istio.io/istio/pkg/test/util/tmpl"
	echot "istio.io/istio/pkg/test/echo"
)

func TestBasic(t *testing.T) {
	// nolint: staticcheck
	framework.NewTest(t).
		RequiresMinClusters(2).
		RequireIstioVersion("1.27").
		Run(func(t framework.TestContext) {
			opt := basicCalls[0]
			opt.To = apps.Captured
			for _, src := range apps.Captured {
				for _, srcWl := range src.WorkloadsOrFail(t) {
					t.NewSubTestf("%v", opt.Scheme).Run(func(t framework.TestContext) {
						src.WithWorkloads(srcWl).CallOrFail(t, opt)
					})
				}
			}
		})
}
