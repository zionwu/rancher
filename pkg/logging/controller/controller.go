package controller

import (
	"github.com/rancher/rancher/pkg/logging/controller/logging"
	"github.com/rancher/types/config"
)

func Register(cluster *config.ClusterContext) {
	logging.RegisterClusterLogging(cluster)
	logging.RegisterProjectLogging(cluster)
	logging.InitData(cluster.K8sClient.CoreV1())
}
