package nad

import (
	"bytes"
	"text/template"

	"github.com/GoogleContainerTools/kpt-functions-sdk/go/fn"
)

type Config struct {
	Name       string
	Namespace  string
	CniVersion string
	CniType    string
	Master     string
	IPPrefix   string
	Gateway    string
}

func GetNadRnode(c *Config) (*fn.KubeObject, error) {
	var nadTemplate = `apiVersion: "k8s.cni.cncf.io/v1"
kind: NetworkAttachmentDefinition
metadata:
  name: {{.Name}}
  namespace: {{.Namespace}}
spec:
  config: '{
		"cniVersion": "{{.CniVersion}}",
	"plugins": [
		{
			"type": "{{.CniType}}",
			"capabilities": { "ips": true },
			"master": "{{.Master}}",
			"mode": "bridge",
			"ipam": {
				"type": "static",
				"addresses": [
          {
            "address": "{{.IPPrefix}}",
            "gateway": "{{.Gateway}}"
          }
        ],
				"routes": [
					{
						"dst": "0.0.0.0/0",
						"gw": "{{.Gateway}}"
					}
				]
			}
		}, {
				"capabilities": { "mac": true },
				"type": "tuning"
		}
	]
	}'
`

	tmpl, err := template.New("nad").Parse(nadTemplate)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]interface{}{
		"Name":       c.Name,
		"Namespace":  c.Namespace,
		"CniVersion": c.CniVersion,
		"CniType":    c.CniType,
		"Master":     c.Master,
		"IPPrefix":   c.IPPrefix,
		"Gateway":    c.Gateway,
	})
	if err != nil {
		return nil, err
	}

	return fn.ParseKubeObject(buf.Bytes())
}
