/*
Copyright 2022 Nokia.

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

package transformer

import (
	"github/henderiw-nephio/nad-inject-fn/pkg/infra"
	"github/henderiw-nephio/nad-inject-fn/pkg/ipam"
	"github/henderiw-nephio/nad-inject-fn/pkg/nad"
	"strings"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
	ipamv1alpha1 "github.com/nokia/k8s-ipam/apis/ipam/v1alpha1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	defaultCniVersion = "0.3.1"
)

// SetNad contains the information to perform the mutator function on a package
type SetNad struct {
	endPoints       map[string]*endPoint
	cniType         string
	masterInterface string
	namespace       string
}

type endPoint struct {
	prefix  string
	gateway string
}

func Run(rl *fn.ResourceList) (bool, error) {
	t := &SetNad{
		endPoints: map[string]*endPoint{},
	}
	// gathers the ip info from the ip-allocations
	t.GatherInfo(rl)

	/*
		for epName, ep := range t.endPoints {
			fmt.Printf("transformData: %s, prefix: %s, gateway: %s\n",
				epName,
				ep.prefix,
				ep.gateway,
			)
		}
	*/
	// transforms the upf with the ip info collected/gathered
	t.GenerateNad(rl)
	return true, nil
}

func (t *SetNad) GatherInfo(rl *fn.ResourceList) {
	for _, o := range rl.Items {
		// parse the node using kyaml
		rn, err := yaml.Parse(o.String())
		if err != nil {
			rl.Results = append(rl.Results, fn.ErrorConfigObjectResult(err, o))
		}
		if rn.GetApiVersion() == "ipam.nephio.org/v1alpha1" && rn.GetKind() == "IPAllocation" {
			if ipam.GetPrefixKind(rn) == string(ipamv1alpha1.PrefixKindNetwork) {
				t.endPoints[rn.GetName()] = &endPoint{
					prefix:  ipam.GetPrefix(rn),
					gateway: ipam.GetGateway(rn),
				}
			}
		}
		if rn.GetApiVersion() == "infra.nephio.io/v1alpha1" && rn.GetKind() == "ClusterContext" {
			t.cniType = infra.GetCniType(rn)
			t.masterInterface = infra.GetMasterInterface(rn)
			t.namespace = rn.GetNamespace()
		}
	}
}

func (t *SetNad) GenerateNad(rl *fn.ResourceList) {

	for epName, ep := range t.endPoints {
		nadNode, err := nad.GetNadRnode(&nad.Config{
			Name:       strings.Join([]string{"upf", epName}, "-"),
			Namespace:  t.namespace,
			CniVersion: defaultCniVersion,
			CniType:    t.cniType,
			Master:     t.masterInterface,
			IPPrefix:   ep.prefix,
			Gateway:    ep.gateway,
		})
		if err != nil {
			rl.Results = append(rl.Results, fn.ErrorConfigObjectResult(err, nadNode))
		}

		rl.Items = append(rl.Items, nadNode)
	}
}
