package tuning

import (
	"context"
	"encoding/base32"
	"fmt"
	log "github.com/ViaQ/logerr/v2/log/static"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	obs "github.com/openshift/cluster-logging-operator/api/observability/v1"
	runtime "github.com/openshift/cluster-logging-operator/internal/runtime"
	obsruntime "github.com/openshift/cluster-logging-operator/internal/runtime/observability"
	"github.com/openshift/cluster-logging-operator/internal/utils"
	framework "github.com/openshift/cluster-logging-operator/test/framework/e2e"
	"github.com/openshift/cluster-logging-operator/test/helpers"
	"github.com/openshift/cluster-logging-operator/test/helpers/oc"
	testruntime "github.com/openshift/cluster-logging-operator/test/runtime/observability"
	"k8s.io/apimachinery/pkg/util/wait"
	"math"
	"time"
)

var _ = Describe("[tuning] deliveryMode AtLeastOnce", func() {

	const (
		componentName  = "log-generator"
		generatorCount = 3
		cpuRequest     = "500m"
		memRequest     = "64Mi"
		linesPerSec    = 500.0
		msgSize        = 256
		forwarderName  = "my-logcollector"
	)

	var (
		e2e         *framework.E2ETestFramework
		receiver    *framework.VectorHttpReceiverLogStore
		err         error
		generatorNS string
		forwarder   *obs.ClusterLogForwarder
	)
	BeforeEach(func() {
		// init the framework
		e2e = framework.NewE2ETestFramework()
		generatorNS = e2e.CreateTestNamespace()
		deployNS := e2e.CreateTestNamespace()

		// deploy receiver
		receiver, err = e2e.DeployHttpReceiver(deployNS)
		Expect(err).To(BeNil())

		sa, err := e2e.BuildAuthorizationFor(deployNS, forwarderName).
			AllowClusterRole(framework.ClusterRoleCollectApplicationLogs).
			Create()
		Expect(err).To(BeNil())

		forwarder = obsruntime.NewClusterLogForwarder(deployNS, forwarderName, runtime.Initialize, func(clf *obs.ClusterLogForwarder) {
			clf.Spec.ServiceAccount.Name = sa.Name
		})

		testruntime.NewClusterLogForwarderBuilder(forwarder).
			FromInputName("myinput", func(spec *obs.InputSpec) {
				spec.Type = obs.InputTypeApplication
				spec.Application = &obs.Application{
					Includes: []obs.NamespaceContainerSpec{
						{Namespace: generatorNS, Container: "log-generator*"},
					},
				}
			}).ToHttpOutput(func(spec *obs.OutputSpec) {
			spec.HTTP.URL = receiver.ClusterLocalEndpoint()
			spec.HTTP.Tuning = &obs.HTTPTuningSpec{
				BaseOutputTuningSpec: obs.BaseOutputTuningSpec{
					DeliveryMode: obs.DeliveryModeAtLeastOnce,
				},
			}
		})

		if err := e2e.CreateObservabilityClusterLogForwarder(forwarder); err != nil {
			Fail(fmt.Sprintf("Unable to create an instance of logforwarder: %v", err))
		}
		if err := e2e.WaitForDaemonSet(forwarder.Namespace, forwarder.Name); err != nil {
			Fail(fmt.Sprintf("Failed waiting for component %s to be ready: %v", helpers.ComponentTypeCollector, err))
		}

		message := []byte{}
		for i := 0; i < msgSize; i++ {
			message = append(message, byte(i))
		}
		delayInMillis := math.Round(1.0 / linesPerSec * 1000.0) //delay to achieve LPS

		// deploy log generator
		options := framework.LogGeneratorOptions{
			Count:          0,
			Delay:          time.Duration(delayInMillis) * time.Millisecond,
			Message:        base32.StdEncoding.EncodeToString(message),
			ContainerCount: generatorCount,
			Labels: map[string]string{
				"testtype":  "myinfra",
				"component": componentName,
			},
		}
		if err := e2e.DeployLogGeneratorWithNamespace(generatorNS, componentName, options); err != nil {
			Fail(fmt.Sprintf("Timed out waiting for the log generator to deploy: %v", err))
		}

	})
	AfterEach(func() {
		e2e.Cleanup()
	})

	It("should deliver all messages even when the collector restarts", func() {

		VerifyCollectedAllLogs := func(timeToWait time.Duration) (totStreams int, duplicates, missing []string) {
			logs, err := receiver.ApplicationLogs(timeToWait)
			Expect(err).To(BeNil())
			Expect(logs).To(Not(BeEmpty()))
			streams := LogStreams{}
			for _, log := range logs {
				Expect(streams.Add(log)).To(Succeed())
			}
			Expect(streams).To(Not(BeEmpty()), "Exp. to extract sequence IDs from the delivered messages")
			streams.Evaluate()
			for _, s := range streams {
				if len(s.Duplicates) != 0 {
					duplicates = append(duplicates, fmt.Sprintf("Exp. to collect messages only once but found %d duplicates for stream %q", len(s.Duplicates), s.Name))
				}
				if len(s.Missing) != 0 {
					missing = append(missing, fmt.Sprintf("Missed %d seqIDs between %d and %d (tot: %d) of stream %q", len(s.Missing), s.First, s.Last, s.Last-s.First, s.Name))
				}
			}
			return streams.Len(), duplicates, missing
		}

		//wait for some logs from all streams to be received
		// Verify some logs from all streams to be received
		Expect(wait.PollUntilContextTimeout(context.TODO(), 5*time.Second, 3*time.Minute, true, func(context.Context) (done bool, err error) {
			q, err := receiver.Query(utils.GetPtr(15 * time.Second))
			if err != nil {
				log.V(0).Error(err, "The error from querying the receiver")
				return true, err
			}
			return len(q.Meta) >= generatorCount, nil
		})).To(Succeed(), "Expected to receive some logs from all log generators before continuing test but did not")
		tot, duplicates, missing := VerifyCollectedAllLogs(30 * time.Second)
		Expect(tot).To(Equal(generatorCount), fmt.Sprintf("Exp. to capture all generator streams: %d/%d", tot, generatorCount))
		Expect(missing).To(BeEmpty())
		Expect(duplicates).To(BeEmpty())

		// Force restart and wait for new pods
		Expect(oc.Literal().From("oc -n %s delete pod -lcomponent=collector", forwarder.Namespace).Output()).To(Succeed())
		if err := e2e.WaitForDaemonSet(forwarder.Namespace, forwarder.Name); err != nil {
			Fail(fmt.Sprintf("Failed waiting for component %s to be ready: %v", helpers.ComponentTypeCollector, err))
		}

		//Verify all logs have been received
		tot, _, missing = VerifyCollectedAllLogs(5 * time.Minute)
		Expect(tot).To(Equal(generatorCount), fmt.Sprintf("Exp. to capture all generator streams: %d/%d", tot, generatorCount))
		Expect(missing).To(BeEmpty())
	})
})
