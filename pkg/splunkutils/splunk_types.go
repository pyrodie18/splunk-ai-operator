package splunkutils



// InstanceType is used to represent the type of Splunk instance (search head, indexer, etc).
type InstanceType string

const (
	// SplunkStandalone is a single instance of Splunk Enterprise
	SplunkStandalone InstanceType = "standalone"

	// SplunkClusterMaster is the manager node of an indexer cluster, see https://docs.splunk.com/Documentation/Splunk/latest/Indexer/Basicclusterarchitecture
	SplunkClusterMaster InstanceType = "cluster-master"

	// SplunkClusterManager is the manager node of an indexer cluster, see https://docs.splunk.com/Documentation/Splunk/latest/Indexer/Basicclusterarchitecture
	SplunkClusterManager InstanceType = "cluster-manager"

	// SplunkSearchHead may be a standalone or clustered search head instance
	SplunkSearchHead InstanceType = "search-head"

	// SplunkIndexer may be a standalone or clustered indexer peer
	SplunkIndexer InstanceType = "indexer"

	// SplunkDeployer is an instance that distributes baseline configurations and apps to search head cluster members
	SplunkDeployer InstanceType = "deployer"

	// SplunkLicenseMaster controls one or more license nodes
	SplunkLicenseMaster InstanceType = "license-master"

	// SplunkLicenseManager controls one or more license nodes
	SplunkLicenseManager InstanceType = "license-manager"

	// SplunkMonitoringConsole is a single instance of Splunk monitor for mc
	SplunkMonitoringConsole InstanceType = "monitoring-console"

	// TmpAppDownloadDir is the Operator directory for app framework, when there is no explicit volume specified
	TmpAppDownloadDir string = "/tmp/appframework/"
)
