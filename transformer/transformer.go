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
	endPoints         map[string]*endPoint
	cniType           string
	masterInterface   string
	namespace         string
	upfDeploymentName string

	existingNads map[string]int // element to kep track of update
}

type endPoint struct {
	prefix  string
	gateway string
}

func Run(rl *fn.ResourceList) (bool, error) {
	t := &SetNad{
		endPoints:    map[string]*endPoint{},
		existingNads: map[string]int{},
	}
	// gathers the ip info from the ip-allocations
	t.GatherInfo(rl)

	//fmt.Printf("cniType: %s\n", t.cniType)
	//fmt.Printf("mastreInterface: %s\n", t.masterInterface)

	// transforms the upf with the ip info collected/gathered
	t.GenerateNad(rl)
	return true, nil
}

func (t *SetNad) GatherInfo(rl *fn.ResourceList) {
	for i, o := range rl.Items {
		// parse the node using kyaml
		rn, err := yaml.Parse(o.String())
		if err != nil {
			rl.Results = append(rl.Results, fn.ErrorConfigObjectResult(err, o))
		}
		if rn.GetApiVersion() == "nf.nephio.org/v1alpha1" && rn.GetKind() == "UPFDeployment" {
			t.upfDeploymentName = rn.GetName()
		}
		if rn.GetApiVersion() == "ipam.nephio.org/v1alpha1" && rn.GetKind() == "IPAllocation" {
			if ipam.GetPrefixKind(rn) == string(ipamv1alpha1.PrefixKindNetwork) {
				t.endPoints[rn.GetLabels()["nephio.org/interface"]] = &endPoint{
					prefix:  ipam.GetPrefix(rn),
					gateway: ipam.GetGateway(rn),
				}
			}
			t.namespace = rn.GetNamespace()
		}
		if rn.GetApiVersion() == "infra.nephio.org/v1alpha1" && rn.GetKind() == "ClusterContext" {
			t.cniType = infra.GetCniType(rn)
			t.masterInterface = infra.GetMasterInterface(rn)
		}
		if rn.GetApiVersion() == "k8s.cni.cncf.io/v1" && rn.GetKind() == "NetworkAttachmentDefinition" {
			t.existingNads[rn.GetName()] = i
		}
	}
}

func (t *SetNad) GenerateNad(rl *fn.ResourceList) {

	for epName, ep := range t.endPoints {
		nadName := strings.Join([]string{t.upfDeploymentName, epName}, "-") // TODO make it a library
		nadNode, err := nad.GetNadRnode(&nad.Config{
			Name:       nadName,
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

		if i, ok := t.existingNads[nadName]; ok {
			// exits -> replace
			rl.Items[i] = nadNode
		} else {
			// add new entry
			rl.Items = append(rl.Items, nadNode)
		}
	}
}
