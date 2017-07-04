@Library('k8s-pipeline-lib') _
 
import com.ft.jenkins.BuildConfig
import com.ft.jenkins.Cluster
 
BuildConfig config = new BuildConfig()
// adjust this to the cluster the application should be deployed to.
// The available values are currently Cluster.DELIVERY, Cluster.PUBLISHING.
// If the app should be deployed in both clusters, put them both in the list
config.deployToClusters = [Cluster.PUBLISHING]
 
entryPointForReleaseAndDev(config)
