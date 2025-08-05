package ambient

import (
	"context"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"

	"istio.io/istio/pkg/test"
	"istio.io/istio/pkg/test/framework"
	"istio.io/istio/pkg/test/framework/components/echo"
	"istio.io/istio/pkg/test/util/retry"
	"istio.io/istio/pilot/pkg/features"
)

func TestGlobalServiceReachability(t *testing.T) {
	test.SetForTest(t, &features.EnableAmbientMultiNetwork, true)

	framework.NewTest(t).
		RequiresMinClusters(2).
		RequireIstioVersion("1.27").
		Run(func(t framework.TestContext) {
			clusters := t.Clusters()
			numClusters := len(clusters)

			clusterToNetwork := make(map[string]string)
			expectedClusters := sets.New[string]()
			expectedNetworks := sets.New[string]()

			for _, c := range clusters {
				name := c.StableName()
				net := c.NetworkName()
				if net == "" {
					net = name
				}
				clusterToNetwork[name] = net
				expectedClusters.Insert(name)
				expectedNetworks.Insert(net)
			}

			for _, svc := range apps.All {
				ns := svc.Config().Namespace.Name()
				svcName := svc.ServiceName()
				for _, c := range clusters {
					if _, err := c.Kube().CoreV1().Services(ns).Get(context.TODO(), svcName, metav1.GetOptions{}); err != nil {
						continue
					}
					if _, err := c.Kube().CoreV1().Services(ns).Patch(
						context.TODO(),
						svcName,
						types.StrategicMergePatchType,
						[]byte(`{"metadata":{"labels":{"istio.io/global":"true"}}}`),
						metav1.PatchOptions{},
					); err != nil {
						t.Fatalf("patch %s/%s in %s: %v", ns, svcName, c.StableName(), err)
					}
				}
			}

			propagation := time.Duration(numClusters*10) * time.Second
			if expectedNetworks.Len() > 1 {
				propagation = time.Duration(numClusters*15) * time.Second
			}
			time.Sleep(propagation)

			testApps := apps.Captured

			maxAttempts := 3
			baseCount := numClusters * 20
			baseTimeout := 10 * time.Second

			for _, dst := range testApps {
				dstName := dst.ServiceName()
				for _, src := range testApps {
					if src.ServiceName() == dstName {
						continue
					}
					for _, wl := range src.WorkloadsOrFail(t) {
						var result echo.CallResult
						var callErr error

						for attempt := 1; attempt <= maxAttempts; attempt++ {
							count := baseCount + (attempt-1)*numClusters*10
							timeout := baseTimeout + time.Duration(attempt-1)*20*time.Second
							callErr = retry.UntilSuccess(func() error {
								var err error
								result, err = src.WithWorkloads(wl).Call(echo.CallOptions{
									To:      dst,
									Port:    echo.Port{Name: "http"},
									Count:   count,
									Timeout: timeout,
								})
								return err
							}, retry.Timeout(timeout+30*time.Second), retry.Delay(5*time.Second))
							if callErr == nil {
								break
							}
							time.Sleep(time.Duration(attempt*attempt*10) * time.Second)
						}
						if callErr != nil {
							t.Fatalf("call failed: %s(%s) -> %s: %v",
								src.ServiceName(), wl.Cluster().StableName(), dstName, callErr)
						}

						respClusters := sets.New[string]()
						respNetworks := sets.New[string]()
						for _, r := range result.Responses {
							respClusters.Insert(r.Cluster)
							if net, ok := clusterToNetwork[r.Cluster]; ok {
								respNetworks.Insert(net)
							}
						}

						if !expectedClusters.Equal(respClusters) {
							missing := expectedClusters.Difference(respClusters).UnsortedList()
							unexpected := respClusters.Difference(expectedClusters).UnsortedList()
							t.Fatalf("cluster mismatch: %s(%s)->%s missing=%v unexpected=%v",
								src.ServiceName(), wl.Cluster().StableName(), dstName, missing, unexpected)
						}

						if !expectedNetworks.Equal(respNetworks) {
							missing := expectedNetworks.Difference(respNetworks).UnsortedList()
							unexpected := respNetworks.Difference(expectedNetworks).UnsortedList()
							t.Fatalf("network mismatch: %s(%s)->%s missing=%v unexpected=%v",
								src.ServiceName(), wl.Cluster().StableName(), dstName, missing, unexpected)
						}
					}
				}
			}
		})
}

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

