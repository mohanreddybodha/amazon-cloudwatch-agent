// Copyright Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT

package awscontainerinsight

import (
	"errors"
	"testing"
	"time"

	"github.com/open-telemetry/opentelemetry-collector-contrib/receiver/awscontainerinsightreceiver"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/confmap"

	"github.com/aws/amazon-cloudwatch-agent/translator/context"
	"github.com/aws/amazon-cloudwatch-agent/translator/translate/otel/common"
)

func TestTranslator(t *testing.T) {
	acit := NewTranslator()
	require.EqualValues(t, "awscontainerinsightreceiver", acit.ID().String())
	testCases := map[string]struct {
		input     map[string]interface{}
		env       map[string]string
		isSystemd bool
		want      *awscontainerinsightreceiver.Config
		wantErr   error
	}{
		"WithoutECSOrKubernetesKeys": {
			input: map[string]interface{}{},
			wantErr: &common.MissingKeyError{
				ID:      acit.ID(),
				JsonKey: "logs::metrics_collected::ecs or logs::metrics_collected::kubernetes",
			},
		},
		"WithECS/WithoutInterval": {
			input: map[string]interface{}{
				"logs": map[string]interface{}{
					"metrics_collected": map[string]interface{}{
						"ecs": map[string]interface{}{},
					},
				},
			},
			want: &awscontainerinsightreceiver.Config{
				ContainerOrchestrator: ecs,
				CollectionInterval:    time.Minute,
				LeaderLockName:        "otel-container-insight-clusterleader",
				TagService:            true,
			},
		},
		"WithECS/WithAgentInterval": {
			input: map[string]interface{}{
				"agent": map[string]interface{}{
					"metrics_collection_interval": float64(20),
				},
				"logs": map[string]interface{}{
					"metrics_collected": map[string]interface{}{
						"ecs": map[string]interface{}{},
					},
				},
			},
			want: &awscontainerinsightreceiver.Config{
				ContainerOrchestrator: ecs,
				CollectionInterval:    20 * time.Second,
				LeaderLockName:        "otel-container-insight-clusterleader",
				TagService:            true,
			},
		},
		"WithECS/WithSectionInterval": {
			input: map[string]interface{}{
				"agent": map[string]interface{}{
					"metrics_collection_interval": float64(20),
				},
				"logs": map[string]interface{}{
					"metrics_collected": map[string]interface{}{
						"ecs": map[string]interface{}{
							"metrics_collection_interval": float64(10),
						},
					},
				},
			},
			want: &awscontainerinsightreceiver.Config{
				ContainerOrchestrator: ecs,
				CollectionInterval:    10 * time.Second,
				LeaderLockName:        "otel-container-insight-clusterleader",
				TagService:            true,
			},
		},
		"WithKubernetes": {
			input: map[string]interface{}{
				"logs": map[string]interface{}{
					"metrics_collected": map[string]interface{}{
						"kubernetes": map[string]interface{}{
							"metrics_collection_interval": float64(10),
							"cluster_name":                "TestCluster",
						},
					},
				},
			},
			want: &awscontainerinsightreceiver.Config{
				ContainerOrchestrator:        eks,
				CollectionInterval:           10 * time.Second,
				ClusterName:                  "TestCluster",
				LeaderLockName:               "cwagent-clusterleader",
				LeaderLockUsingConfigMapOnly: true,
				TagService:                   true,
				KubeConfigPath:               "",
			},
		},
		"WithKubernetes/WithoutClusterName": {
			input: map[string]interface{}{
				"logs": map[string]interface{}{
					"metrics_collected": map[string]interface{}{
						"kubernetes": map[string]interface{}{},
					},
				},
			},
			wantErr: errors.New("cluster name is not provided and was not auto-detected from EC2 tags"),
		},
		"WithKubernetes/WithTagService": {
			input: map[string]interface{}{
				"logs": map[string]interface{}{
					"metrics_collected": map[string]interface{}{
						"kubernetes": map[string]interface{}{
							"tag_service":  false,
							"cluster_name": "TestCluster",
						},
					},
				},
			},
			want: &awscontainerinsightreceiver.Config{
				ContainerOrchestrator:        eks,
				CollectionInterval:           60 * time.Second,
				TagService:                   false,
				LeaderLockName:               defaultLeaderLockName,
				LeaderLockUsingConfigMapOnly: true,
				ClusterName:                  "TestCluster",
				KubeConfigPath:               "",
			},
		},
		"WithKubernetes/WithCollectionRoleLeader": {
			input: map[string]interface{}{
				"logs": map[string]interface{}{
					"metrics_collected": map[string]interface{}{
						"kubernetes": map[string]interface{}{
							"cluster_name": "TestCluster",
						},
					},
				},
			},
			env: map[string]string{
				"CWAGENT_ROLE": "LEADER",
			},
			want: &awscontainerinsightreceiver.Config{
				ContainerOrchestrator:        eks,
				CollectionInterval:           60 * time.Second,
				CollectionRole:               awscontainerinsightreceiver.LEADER,
				TagService:                   true,
				LeaderLockName:               defaultLeaderLockName,
				LeaderLockUsingConfigMapOnly: true,
				ClusterName:                  "TestCluster",
				KubeConfigPath:               "",
			},
		},
		"WithKubernetes/WithCollectionRoleNode": {
			input: map[string]interface{}{
				"logs": map[string]interface{}{
					"metrics_collected": map[string]interface{}{
						"kubernetes": map[string]interface{}{
							"cluster_name": "TestCluster",
						},
					},
				},
			},
			env: map[string]string{
				"CWAGENT_ROLE": "NODE",
			},
			want: &awscontainerinsightreceiver.Config{
				ContainerOrchestrator:        eks,
				CollectionInterval:           60 * time.Second,
				CollectionRole:               awscontainerinsightreceiver.NODE,
				TagService:                   true,
				LeaderLockName:               defaultLeaderLockName,
				LeaderLockUsingConfigMapOnly: true,
				ClusterName:                  "TestCluster",
				KubeConfigPath:               "",
			},
		},
		"WithKubernetes/WithEnhancedContainerInsights": {
			input: map[string]interface{}{
				"logs": map[string]interface{}{
					"metrics_collected": map[string]interface{}{
						"kubernetes": map[string]interface{}{
							"enhanced_container_insights": true,
							"cluster_name":                "TestCluster",
						},
					},
				},
			},
			want: &awscontainerinsightreceiver.Config{
				ContainerOrchestrator:        eks,
				CollectionInterval:           60 * time.Second,
				PrefFullPodName:              true,
				LeaderLockName:               defaultLeaderLockName,
				LeaderLockUsingConfigMapOnly: true,
				ClusterName:                  "TestCluster",
				TagService:                   true,
				EnableControlPlaneMetrics:    true,
				AddFullPodNameMetricLabel:    true,
				AddContainerNameMetricLabel:  true,
				KubeConfigPath:               "",
			},
		},
		"WithKubernetes/WithLevel1Granularity": {
			input: map[string]interface{}{
				"logs": map[string]interface{}{
					"metrics_collected": map[string]interface{}{
						"kubernetes": map[string]interface{}{
							"metric_granularity": 1,
							"cluster_name":       "TestCluster",
						},
					},
				},
			},
			want: &awscontainerinsightreceiver.Config{
				ContainerOrchestrator:        eks,
				CollectionInterval:           60 * time.Second,
				LeaderLockName:               defaultLeaderLockName,
				LeaderLockUsingConfigMapOnly: true,
				ClusterName:                  "TestCluster",
				TagService:                   true,
				EnableControlPlaneMetrics:    false,
				AddFullPodNameMetricLabel:    false,
				AddContainerNameMetricLabel:  false,
				KubeConfigPath:               "",
			},
		},
		"WithKubernetes/WithLevel2Granularity": {
			input: map[string]interface{}{
				"logs": map[string]interface{}{
					"metrics_collected": map[string]interface{}{
						"kubernetes": map[string]interface{}{
							"metric_granularity": 2,
							"cluster_name":       "TestCluster",
						},
					},
				},
			},
			want: &awscontainerinsightreceiver.Config{
				ContainerOrchestrator:        eks,
				CollectionInterval:           60 * time.Second,
				PrefFullPodName:              true,
				LeaderLockName:               defaultLeaderLockName,
				LeaderLockUsingConfigMapOnly: true,
				ClusterName:                  "TestCluster",
				TagService:                   true,
				EnableControlPlaneMetrics:    true,
				AddFullPodNameMetricLabel:    true,
				AddContainerNameMetricLabel:  true,
				KubeConfigPath:               "",
			},
		},
		"WithKubernetes/WithLevel3Granularity": {
			input: map[string]interface{}{
				"logs": map[string]interface{}{
					"metrics_collected": map[string]interface{}{
						"kubernetes": map[string]interface{}{
							"metric_granularity": 3,
							"cluster_name":       "TestCluster",
						},
					},
				},
			},
			want: &awscontainerinsightreceiver.Config{
				ContainerOrchestrator:        eks,
				CollectionInterval:           60 * time.Second,
				PrefFullPodName:              true,
				LeaderLockName:               defaultLeaderLockName,
				LeaderLockUsingConfigMapOnly: true,
				ClusterName:                  "TestCluster",
				TagService:                   true,
				EnableControlPlaneMetrics:    true,
				AddFullPodNameMetricLabel:    true,
				AddContainerNameMetricLabel:  true,
				KubeConfigPath:               "",
			},
		},
		"WithECSAndKubernetes": {
			input: map[string]interface{}{
				"logs": map[string]interface{}{
					"metrics_collected": map[string]interface{}{
						"ecs": map[string]interface{}{
							"metrics_collection_interval": float64(5),
						},
						"kubernetes": map[string]interface{}{
							"metrics_collection_interval": float64(10),
						},
					},
				},
			},
			want: &awscontainerinsightreceiver.Config{
				ContainerOrchestrator:        ecs,
				CollectionInterval:           5 * time.Second,
				LeaderLockName:               "otel-container-insight-clusterleader",
				LeaderLockUsingConfigMapOnly: false,
				TagService:                   true,
			},
		},
		"WithEKSAndCustomKubeConfigPathHostDetails": {
			input: map[string]interface{}{
				"logs": map[string]interface{}{
					"metrics_collected": map[string]interface{}{
						"kubernetes": map[string]interface{}{
							"kube_config_path": "/tmp/custom.kubeconfig",
							"cluster_name":     "TestCluster",
							"host_name":        "test-hostname",
							"host_ip":          "1.2.3.4",
						},
					},
				},
			},
			isSystemd: true,
			want: &awscontainerinsightreceiver.Config{
				ContainerOrchestrator:        eks,
				CollectionInterval:           60 * time.Second,
				PrefFullPodName:              false,
				LeaderLockName:               defaultLeaderLockName,
				LeaderLockUsingConfigMapOnly: true,
				ClusterName:                  "TestCluster",
				TagService:                   true,
				KubeConfigPath:               "/tmp/custom.kubeconfig",
				HostName:                     "test-hostname",
				HostIP:                       "1.2.3.4",
				RunOnSystemd:                 true,
			},
		},
	}
	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			context.CurrentContext().SetRunInContainer(!testCase.isSystemd)
			conf := confmap.NewFromStringMap(testCase.input)
			got, err := acit.Translate(conf)
			require.Equal(t, testCase.wantErr, err)
			if testCase.env != nil {
				for k, v := range testCase.env {
					t.Setenv(k, v)
				}
			}
			if err == nil {
				require.NotNil(t, got)
				gotCfg, ok := got.(*awscontainerinsightreceiver.Config)
				require.True(t, ok)
				require.Equal(t, testCase.want.ContainerOrchestrator, gotCfg.ContainerOrchestrator)
				require.Equal(t, testCase.want.CollectionInterval, gotCfg.CollectionInterval)
				require.Equal(t, testCase.want.PrefFullPodName, gotCfg.PrefFullPodName)
				require.Equal(t, testCase.want.ClusterName, gotCfg.ClusterName)
				require.Equal(t, testCase.want.AddContainerNameMetricLabel, gotCfg.AddContainerNameMetricLabel)
				require.Equal(t, testCase.want.AddFullPodNameMetricLabel, gotCfg.AddFullPodNameMetricLabel)
				require.Equal(t, testCase.want.TagService, gotCfg.TagService)
				require.Equal(t, testCase.want.LeaderLockName, gotCfg.LeaderLockName)
				require.Equal(t, testCase.want.LeaderLockUsingConfigMapOnly, gotCfg.LeaderLockUsingConfigMapOnly)
				require.Equal(t, testCase.want.EnableControlPlaneMetrics, gotCfg.EnableControlPlaneMetrics)
				require.Equal(t, testCase.want.KubeConfigPath, gotCfg.KubeConfigPath)
				require.Equal(t, testCase.want.HostName, gotCfg.HostName)
				require.Equal(t, testCase.want.HostIP, gotCfg.HostIP)
				require.Equal(t, testCase.want.RunOnSystemd, gotCfg.RunOnSystemd)
			}
		})
	}
}
