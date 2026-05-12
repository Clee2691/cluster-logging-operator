package azurelogsingestion

import (
	obs "github.com/openshift/cluster-logging-operator/api/observability/v1"
	"github.com/openshift/cluster-logging-operator/internal/api/observability"
	. "github.com/openshift/cluster-logging-operator/internal/generator/framework"
	genhelper "github.com/openshift/cluster-logging-operator/internal/generator/helpers"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/elements"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/helpers"
	vectorhelpers "github.com/openshift/cluster-logging-operator/internal/generator/vector/helpers"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/output/common"
	"github.com/openshift/cluster-logging-operator/internal/generator/vector/output/common/tls"
)

const (
	azureCredentialKindClientSecret = "client_secret_credential"
)

type AzureLogsIngestion struct {
	ComponentID       string
	Inputs            string
	Endpoint          string
	DcrImmutableId    string
	StreamName        string
	TokenScope        string
	TimestampField    string
	CredentialKind    string
	AzureTenantId     string
	AzureClientId     string
	AzureClientSecret string
}

func (a AzureLogsIngestion) Name() string {
	return "azureLogsIngestionTemplate"
}

func (a AzureLogsIngestion) Template() string {
	return `{{define "` + a.Name() + `" -}}
[sinks.{{.ComponentID}}]
type = "azure_logs_ingestion"
inputs = {{.Inputs}}
endpoint = "{{.Endpoint}}"
dcr_immutable_id = "{{.DcrImmutableId}}"
stream_name = "{{.StreamName}}"
{{ if .TokenScope -}}
token_scope = "{{.TokenScope}}"
{{ end -}}
{{ if .TimestampField -}}
timestamp_field = "{{.TimestampField}}"
{{ end -}}

[sinks.{{.ComponentID}}.auth]
azure_credential_kind = "{{.CredentialKind}}"
azure_tenant_id = "{{.AzureTenantId}}"
azure_client_id = "{{.AzureClientId}}"
azure_client_secret = "{{.AzureClientSecret}}"
{{end}}`
}

func New(id string, o obs.OutputSpec, inputs []string, secrets observability.Secrets, strategy common.ConfigStrategy, op Options) []Element {
	if genhelper.IsDebugOutput(op) {
		return []Element{
			elements.Debug(helpers.MakeID(id, "debug"), vectorhelpers.MakeInputs(inputs...)),
		}
	}
	azli := o.AzureLogsIngestion
	e := AzureLogsIngestion{
		ComponentID:    id,
		Inputs:         vectorhelpers.MakeInputs(inputs...),
		Endpoint:       azli.URL,
		DcrImmutableId: azli.DcrImmutableId,
		StreamName:     azli.StreamName,
		TokenScope:     azli.TokenScope,
		TimestampField: azli.TimestampField,
		CredentialKind: azureCredentialKindClientSecret,
	}
	if azli.Authentication != nil && azli.Authentication.ClientSecret != nil {
		cs := azli.Authentication.ClientSecret
		e.AzureTenantId = cs.TenantId
		e.AzureClientId = cs.ClientId
		if cs.Secret != nil {
			e.AzureClientSecret = vectorhelpers.SecretFrom(cs.Secret)
		}
	}
	return MergeElements(
		[]Element{},
		[]Element{
			e,
			common.NewEncoding(id, ""),
			common.NewAcknowledgments(id, strategy),
			common.NewBatch(id, strategy),
			common.NewBuffer(id, strategy),
			common.NewRequest(id, strategy),
			tls.New(id, o.TLS, secrets, op),
		},
	)
}
