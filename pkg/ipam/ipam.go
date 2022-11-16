package ipam

import (
	"strings"

	"sigs.k8s.io/kustomize/kyaml/utils"
	kyaml "sigs.k8s.io/kustomize/kyaml/yaml"
)

func GetValue(source *kyaml.RNode, fp string) string {
	fieldPath := utils.SmarterPathSplitter(fp, ".")
	foundValue, lookupErr := source.Pipe(&kyaml.PathGetter{Path: fieldPath})
	if lookupErr != nil {
		return ""
	}
	return strings.TrimSuffix(foundValue.MustString(), "\n")
}

func GetPrefixKind(source *kyaml.RNode) string {
	return GetValue(source, "spec.kind")
}

func GetGateway(source *kyaml.RNode) string {
	return GetValue(source, "status.gateway")
}

func GetPrefix(source *kyaml.RNode) string {
	return GetValue(source, "status.allocatedprefix")
}
