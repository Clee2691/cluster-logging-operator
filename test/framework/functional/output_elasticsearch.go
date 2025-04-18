package functional

import (
	"strconv"
	"strings"

	obs "github.com/openshift/cluster-logging-operator/api/observability/v1"

	"github.com/openshift/cluster-logging-operator/internal/generator/url"
	"github.com/openshift/cluster-logging-operator/internal/runtime"

	log "github.com/ViaQ/logerr/v2/log/static"
)

type ElasticsearchVersion int

const (
	ElasticsearchVersion6 ElasticsearchVersion = 6
	ElasticsearchVersion7 ElasticsearchVersion = 7
	ElasticsearchVersion8 ElasticsearchVersion = 8
)

var (
	esVersionToImage = map[ElasticsearchVersion]string{
		ElasticsearchVersion6: "elasticsearch:6.8.23",
		ElasticsearchVersion7: "elasticsearch:7.10.1",
		ElasticsearchVersion8: "elasticsearch:8.6.1",
	}
)

func (f *CollectorFunctionalFramework) AddES7Output(b *runtime.PodBuilder, output obs.OutputSpec) error {
	return AddESOutput(ElasticsearchVersion7, b, output)
}

func AddESOutput(version ElasticsearchVersion, b *runtime.PodBuilder, output obs.OutputSpec) error {
	log.V(2).Info("Adding elasticsearch output", "name", output.Name, "version", version)
	name := strings.ToLower(output.Name)

	esURL, err := url.Parse(output.Elasticsearch.URL)
	if err != nil {
		return err
	}
	transportPort, portError := determineTransportPort(esURL.Port())
	if portError != nil {
		return portError
	}

	log.V(2).Info("Adding container", "name", name)
	log.V(2).Info("Adding ElasticSearch output container", "name", obs.OutputTypeElasticsearch)

	b.AddContainer(name, esVersionToImage[version]).
		AddEnvVar("discovery.type", "single-node").
		AddEnvVar("http.port", esURL.Port()).
		AddEnvVar("transport.port", transportPort).
		AddEnvVar("xpack.security.enabled", "false").
		AddRunAsUser(2000).
		End()
	return nil
}

func determineTransportPort(port string) (string, error) {
	iPort, err := strconv.Atoi(port)
	if err != nil {
		return "", err
	}
	iPort = iPort + 100
	return strconv.Itoa(iPort), nil
}
