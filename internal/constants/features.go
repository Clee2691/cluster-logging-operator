package constants

const (
	Enabled = "enabled"

	// UseOldRemoteSyslogPlugin use old syslog plugin (docebo/fluent-plugin-remote-syslog)
	// +deprecated
	UseOldRemoteSyslogPlugin = "clusterlogging.openshift.io/useoldremotesyslogplugin"

	AnnotationDebugOutput = "logging.openshift.io/debug-output"

	// AnnotationEnableSchema is the annotation to enable alternate output formats of logs.
	// Currently only viaq & opentelemetry are supported
	AnnotationEnableSchema = "logging.openshift.io/enableschema"

	// AnnotationOCPConsoleMigrationTarget is to be used to enable the OCP Console for Logs
	// without switching the default `logStore` to LokiStack. The value should be the
	// LokiStack resource name representing the target store for the migration.
	AnnotationOCPConsoleMigrationTarget = "logging.openshift.io/force-enable-ocp-console-target"

	// AnnotationEnableCollectorAsDeployment is to enable deploying the collector as a deployment
	// instead of a daemonset to support the HCP use case of using the collector for collecting
	// audit logs via a webhook.
	AnnotationEnableCollectorAsDeployment = "logging.openshift.io/enable-collector-as-deployment"
)
