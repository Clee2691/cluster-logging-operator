package functional

import (
	"fmt"
	"github.com/openshift/cluster-logging-operator/internal/certificates"
	"github.com/openshift/cluster-logging-operator/internal/runtime"
	testruntime "github.com/openshift/cluster-logging-operator/test/runtime"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	yaml "sigs.k8s.io/yaml"

	"github.com/ViaQ/logerr/log"
	logging "github.com/openshift/cluster-logging-operator/apis/logging/v1"
	"github.com/openshift/cluster-logging-operator/internal/constants"
	"github.com/openshift/cluster-logging-operator/internal/pkg/generator/forwarder"
	"github.com/openshift/cluster-logging-operator/internal/utils"
	"github.com/openshift/cluster-logging-operator/test"
	"github.com/openshift/cluster-logging-operator/test/client"
	frameworkfluent "github.com/openshift/cluster-logging-operator/test/framework/functional/fluentd"
	"github.com/openshift/cluster-logging-operator/test/helpers/oc"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type receiverBuilder func(f *CollectorFunctionalFramework, b *runtime.PodBuilder, output logging.OutputSpec) error

type CollectorFramework interface {
	DeployConfigMapForConfig(name, config, clfYaml string) error
	BuildCollectorContainer(*runtime.ContainerBuilder, string) *runtime.ContainerBuilder
	IsStarted(string) bool
}

//CollectorFunctionalFramework deploys stand alone fluentd with the fluent.conf as generated by input ClusterLogForwarder CR
type CollectorFunctionalFramework struct {
	Name              string
	Namespace         string
	Conf              string
	image             string
	Labels            map[string]string
	Forwarder         *logging.ClusterLogForwarder
	Test              *client.Test
	Pod               *corev1.Pod
	fluentContainerId string
	receiverBuilders  []receiverBuilder
	closeClient       func()

	collector CollectorFramework
}

func NewCollectorFunctionalFramework() *CollectorFunctionalFramework {
	test := client.NewTest()
	return NewCollectorFunctionalFrameworkUsing(test, test.Close, 0)
}

func NewFluentdFunctionalFrameworkForTest(t *testing.T) *CollectorFunctionalFramework {
	return NewCollectorFunctionalFrameworkUsing(client.ForTest(t), func() {}, 0)
}

func NewCollectorFunctionalFrameworkUsing(t *client.Test, fnClose func(), verbosity int) *CollectorFunctionalFramework {
	if level, found := os.LookupEnv("LOG_LEVEL"); found {
		if i, err := strconv.Atoi(level); err == nil {
			verbosity = i
		}
	}

	log.MustInit("functional-framework")
	log.SetLogLevel(verbosity)
	testName := "functional"
	framework := &CollectorFunctionalFramework{
		Name:      testName,
		Namespace: t.NS.Name,
		image:     utils.GetComponentImage(constants.FluentdName),
		Labels: map[string]string{
			"testtype": "functional",
			"testname": testName,
		},
		Test:             t,
		Forwarder:        testruntime.NewClusterLogForwarder(),
		receiverBuilders: []receiverBuilder{},
		closeClient:      fnClose,
		collector: &frameworkfluent.FluentdCollector{
			Test: t,
		},
	}
	framework.Forwarder.SetNamespace(t.NS.Name)
	return framework
}

func (f *CollectorFunctionalFramework) Cleanup() {
	f.closeClient()
}

func (f *CollectorFunctionalFramework) RunCommand(container string, cmd ...string) (string, error) {
	log.V(2).Info("Running", "container", container, "cmd", cmd)
	out, err := testruntime.ExecOc(f.Pod, strings.ToLower(container), cmd[0], cmd[1:]...)
	log.V(2).Info("Exec'd", "out", out, "err", err)
	return out, err
}

func (f *CollectorFunctionalFramework) AddOutputContainersVisitors() []runtime.PodBuilderVisitor {
	visitors := []runtime.PodBuilderVisitor{
		func(b *runtime.PodBuilder) error {
			return f.addOutputContainers(b, f.Forwarder.Spec.Outputs)
		},
	}
	return visitors
}

//Deploy the objects needed to functional Test
func (f *CollectorFunctionalFramework) Deploy() (err error) {
	return f.DeployWithVisitors(f.AddOutputContainersVisitors())
}

func (f *CollectorFunctionalFramework) DeployWithVisitor(visitor runtime.PodBuilderVisitor) (err error) {
	visitors := []runtime.PodBuilderVisitor{
		visitor,
	}
	return f.DeployWithVisitors(visitors)
}

//Deploy the objects needed to functional Test
func (f *CollectorFunctionalFramework) DeployWithVisitors(visitors []runtime.PodBuilderVisitor) (err error) {
	log.V(2).Info("Generating config", "forwarder", f.Forwarder)
	clfYaml, _ := yaml.Marshal(f.Forwarder)
	debugOutput := false
	testClient := client.Get().ControllerRuntimeClient()
	if strings.TrimSpace(f.Conf) == "" {
		if f.Conf, err = forwarder.Generate(string(clfYaml), false, debugOutput, &testClient); err != nil {
			return err
		}
	} else {
		log.V(2).Info("Using provided collector conf instead of generating one")
	}

	if err = f.collector.DeployConfigMapForConfig(f.Name, f.Conf, string(clfYaml)); err != nil {
		return err
	}

	log.V(2).Info("Generating Certificates")
	if err, _, _ = certificates.GenerateCertificates(f.Test.NS.Name,
		test.GitRoot("scripts"), "elasticsearch",
		utils.DefaultWorkingDir); err != nil {
		return err
	}
	log.V(2).Info("Creating certs configmap")
	certsName := "certs-" + f.Name
	certs := runtime.NewConfigMap(f.Test.NS.Name, certsName, map[string]string{})
	runtime.NewConfigMapBuilder(certs).
		Add("tls.key", string(utils.GetWorkingDirFileContents("system.logging.fluentd.key"))).
		Add("tls.crt", string(utils.GetWorkingDirFileContents("system.logging.fluentd.crt")))
	if err = f.Test.Client.Create(certs); err != nil {
		return err
	}

	log.V(2).Info("Creating service")
	service := runtime.NewService(f.Test.NS.Name, f.Name)
	runtime.NewServiceBuilder(service).
		AddServicePort(24231, 24231).
		WithSelector(f.Labels)
	if err = f.Test.Client.Create(service); err != nil {
		return err
	}

	role := runtime.NewRole(f.Test.NS.Name, f.Name,
		v1.PolicyRule{
			Verbs:     []string{"list", "get"},
			Resources: []string{"pods", "namespaces"},
			APIGroups: []string{""},
		},
	)
	if err = f.Test.Client.Create(role); err != nil {
		return err
	}
	rolebinding := runtime.NewRoleBinding(f.Test.NS.Name, f.Name,
		v1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     role.Name,
		},
		v1.Subject{
			Kind: "ServiceAccount",
			Name: "default",
		},
	)
	if err = f.Test.Client.Create(rolebinding); err != nil {
		return err
	}

	log.V(2).Info("Defining pod...")
	f.Pod = runtime.NewPod(f.Test.NS.Name, f.Name)
	b := runtime.NewPodBuilder(f.Pod).
		WithLabels(f.Labels).
		AddConfigMapVolume("config", f.Name).
		AddConfigMapVolume("entrypoint", f.Name).
		AddConfigMapVolume("certs", certsName)
	b = f.collector.BuildCollectorContainer(b.AddContainer(constants.CollectorName, f.image), FunctionalNodeName).End()

	for _, visit := range visitors {
		if err = visit(b); err != nil {
			return err
		}
	}
	log.V(2).Info("Creating pod", "pod", f.Pod)
	if err = f.Test.Client.Create(f.Pod); err != nil {
		return err
	}

	log.V(2).Info("waiting for pod to be ready")
	if err = oc.Literal().From("oc wait -n %s pod/%s --timeout=120s --for=condition=Ready", f.Test.NS.Name, f.Name).Output(); err != nil {
		return err
	}
	if err = f.Test.Client.Get(f.Pod); err != nil {
		return err
	}
	log.V(2).Info("waiting for service endpoints to be ready")
	err = wait.PollImmediate(time.Second*2, time.Second*10, func() (bool, error) {
		ips, err := oc.Get().WithNamespace(f.Test.NS.Name).Resource("endpoints", f.Name).OutputJsonpath("{.subsets[*].addresses[*].ip}").Run()
		if err != nil {
			return false, nil
		}
		// if there are IPs in the service endpoint, the service is available
		if strings.TrimSpace(ips) != "" {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("service could not be started")
	}
	log.V(2).Info("waiting for the collector to be ready")
	err = wait.PollImmediate(time.Second*2, time.Second*30, func() (bool, error) {
		output, err := oc.Literal().From("oc logs -n %s pod/%s -c %s", f.Test.NS.Name, f.Name, constants.CollectorName).Run()
		if err != nil {
			return false, nil
		}

		// if fluentd started successfully return success
		if f.collector.IsStarted(output) || debugOutput {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return fmt.Errorf("collector did not start in the container")
	}
	for _, cs := range f.Pod.Status.ContainerStatuses {
		if cs.Name == constants.CollectorName {
			f.fluentContainerId = strings.TrimPrefix(cs.ContainerID, "cri-o://")
			break
		}
	}
	return nil
}

func (f *CollectorFunctionalFramework) addOutputContainers(b *runtime.PodBuilder, outputs []logging.OutputSpec) error {
	log.V(2).Info("Adding outputs", "outputs", outputs)
	for _, output := range outputs {
		switch output.Type {
		case logging.OutputTypeFluentdForward:
			if err := f.AddForwardOutput(b, output); err != nil {
				return err
			}
		case logging.OutputTypeSyslog:
			if err := f.addSyslogOutput(b, output); err != nil {
				return err
			}
		case logging.OutputTypeKafka:
			if err := f.addKafkaOutput(b, output); err != nil {
				return err
			}
		case logging.OutputTypeElasticsearch:
			if err := f.addES7Output(b, output); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *CollectorFunctionalFramework) WaitForPodToBeReady() error {
	return oc.Literal().From("oc wait -n %s pod/%s --timeout=60s --for=condition=Ready", f.Test.NS.Name, f.Name).Output()
}
