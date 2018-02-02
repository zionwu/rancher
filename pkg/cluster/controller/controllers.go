package controller

import (
	"context"

	alertController "github.com/rancher/rancher/pkg/alert/controller"
	"github.com/rancher/rancher/pkg/cluster/controller/authz"
	"github.com/rancher/rancher/pkg/cluster/controller/eventssyncer"
	"github.com/rancher/rancher/pkg/cluster/controller/healthsyncer"
	"github.com/rancher/rancher/pkg/cluster/controller/nodesyncer"
	"github.com/rancher/rancher/pkg/cluster/controller/secret"
	helmController "github.com/rancher/rancher/pkg/helm/controller"
	loggingController "github.com/rancher/rancher/pkg/logging/controller"
	workloadController "github.com/rancher/rancher/pkg/workload/controller"

	"github.com/rancher/types/config"
)

func Register(ctx context.Context, cluster *config.ClusterContext) error {
	nodesyncer.Register(cluster)
	healthsyncer.Register(ctx, cluster)
	authz.Register(cluster)
	eventssyncer.Register(cluster)
	secret.Register(cluster)
	helmController.Register(cluster)
	loggingController.Register(cluster)
	alertController.Register(ctx, cluster)
	workloadContext := cluster.WorkloadContext()
	return workloadController.Register(ctx, workloadContext)
}
