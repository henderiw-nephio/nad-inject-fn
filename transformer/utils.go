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
	"fmt"
	"strings"

	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/resid"
	"sigs.k8s.io/kustomize/kyaml/utils"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// MakeResIds returns all of an RNode's Ids
func MakeResIds(n *yaml.RNode) ([]resid.ResId, error) {
	var result []resid.ResId
	apiVersion := n.Field(yaml.APIVersionField)
	var group, version string
	if apiVersion != nil {
		group, version = resid.ParseGroupVersion(yaml.GetValue(apiVersion.Value))
	}
	result = append(result, resid.NewResIdWithNamespace(
		resid.Gvk{Group: group, Version: version, Kind: n.GetKind()}, n.GetName(), n.GetNamespace()),
	)
	return result, nil
}

func GetPrefixFromIPAlloc(source *yaml.RNode) (*yaml.RNode, error) {
	fp := "status.prefix"
	fieldPath := utils.SmarterPathSplitter(fp, ".")
	foundValue, lookupErr := source.Pipe(&yaml.PathGetter{Path: fieldPath})
	if lookupErr != nil {
		return nil, fmt.Errorf("error finding field in target: %w", lookupErr)
	}
	return foundValue, nil
}

func GetGatewayFromIPAlloc(source *yaml.RNode) (*yaml.RNode, error) {
	fp := "status.gateway"
	fieldPath := utils.SmarterPathSplitter(fp, ".")
	foundValue, lookupErr := source.Pipe(&yaml.PathGetter{Path: fieldPath})
	if lookupErr != nil {
		return nil, fmt.Errorf("error finding field in target: %w", lookupErr)
	}
	return foundValue, nil
}

func CopyValueToTarget(target *yaml.RNode, value *yaml.RNode, selector *types.TargetSelector) error {
	for _, fp := range selector.FieldPaths {
		fieldPath := utils.SmarterPathSplitter(fp, ".")
		create, err := ShouldCreateField(selector.Options, fieldPath)
		if err != nil {
			return err
		}

		//fmt.Printf("fieldPath: %v\n", fieldPath)
		var targetFields []*yaml.RNode
		if create {
			createdField, createErr := target.Pipe(yaml.LookupCreate(value.YNode().Kind, fieldPath...))
			if createErr != nil {
				return fmt.Errorf("error creating node: %w", createErr)
			}
			targetFields = append(targetFields, createdField)
		} else {
			// may return multiple fields, always wrapped in a sequence node
			foundFieldSequence, lookupErr := target.Pipe(&yaml.PathMatcher{Path: fieldPath})
			if lookupErr != nil {
				return fmt.Errorf("error finding field in target: %w", lookupErr)
			}
			targetFields, err = foundFieldSequence.Elements()
			if err != nil {
				return fmt.Errorf("error fetching elements in replacement target: %w", err)
			}
		}

		for _, t := range targetFields {
			if err := SetFieldValue(selector.Options, t, value); err != nil {
				return err
			}
		}

	}
	return nil
}

func SetFieldValue(options *types.FieldOptions, targetField *yaml.RNode, value *yaml.RNode) error {
	//fmt.Printf("setFieldValue options: %v\n", options)
	//fmt.Printf("setFieldValue targetField: %v\n", targetField.MustString())
	//fmt.Printf("setFieldValue value: %v\n", value.MustString())
	value = value.Copy()
	if options != nil && options.Delimiter != "" {
		if targetField.YNode().Kind != yaml.ScalarNode {
			return fmt.Errorf("delimiter option can only be used with scalar nodes")
		}
		tv := strings.Split(targetField.YNode().Value, options.Delimiter)
		v := yaml.GetValue(value)
		// TODO: Add a way to remove an element
		switch {
		case options.Index < 0: // prefix
			tv = append([]string{v}, tv...)
		case options.Index >= len(tv): // suffix
			tv = append(tv, v)
		default: // replace an element
			tv[options.Index] = v
		}
		value.YNode().Value = strings.Join(tv, options.Delimiter)
	}

	if targetField.YNode().Kind == yaml.ScalarNode {
		// For scalar, only copy the value (leave any type intact to auto-convert int->string or string->int)
		targetField.YNode().Value = value.YNode().Value
	} else {
		targetField.SetYNode(value.YNode())
	}

	return nil
}

func ShouldCreateField(options *types.FieldOptions, fieldPath []string) (bool, error) {
	if options == nil || !options.Create {
		return false, nil
	}
	// create option is not supported in a wildcard matching
	for _, f := range fieldPath {
		if f == "*" {
			return false, fmt.Errorf("cannot support create option in a multi-value target")
		}
	}
	return true, nil
}
