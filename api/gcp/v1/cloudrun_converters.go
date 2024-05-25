package v1

import (
	"fmt"

	"cloud.google.com/go/run/apiv2/runpb"
)

func (c *CloudRun) ConvertToCreateServiceRequest() *runpb.CreateServiceRequest {
	return &runpb.CreateServiceRequest{
		Parent:    fmt.Sprintf("projects/%s/locations/%s", c.Spec.ProjectID, c.Spec.Location),
		ServiceId: fmt.Sprintf("%s-%s", c.Namespace, c.Name),
		Service:   c.convertToService(),
	}
}

func (c *CloudRun) GetGcpCloudRunServiceFullName() string {
	return fmt.Sprintf("projects/%s/locations/%s/services/%s-%s", c.Spec.ProjectID, c.Spec.Location, c.Namespace, c.Name)
}

func (c *CloudRun) convertToService() *runpb.Service {
	return &runpb.Service{
		Ingress: c.Spec.TrafficMode,
		Traffic: c.convertToTrafficTarget(),
		Template: &runpb.RevisionTemplate{
			Containers: c.convertToContainers(),
		},
	}
}

func (c *CloudRun) convertToTrafficTarget() []*runpb.TrafficTarget {
	var trafficTargets []*runpb.TrafficTarget
	for _, traffic := range c.Spec.Traffic {
		if traffic.LatestRevision {
			trafficTargets = append(trafficTargets, &runpb.TrafficTarget{
				Percent: traffic.Percent,
				Type:    runpb.TrafficTargetAllocationType_TRAFFIC_TARGET_ALLOCATION_TYPE_LATEST,
			})
		} else {
			trafficTargets = append(trafficTargets, &runpb.TrafficTarget{
				Percent:  traffic.Percent,
				Revision: traffic.Revision,
				Type:     runpb.TrafficTargetAllocationType_TRAFFIC_TARGET_ALLOCATION_TYPE_REVISION,
			})
		}
	}
	return trafficTargets
}

func (c *CloudRun) convertToContainers() []*runpb.Container {
	containers := make([]*runpb.Container, 0, len(c.Spec.Containers))
	for _, container := range c.Spec.Containers {
		containers = append(containers, &runpb.Container{
			Image: container.Image,
			Name:  container.Name,
			Ports: []*runpb.ContainerPort{
				{
					ContainerPort: container.Port,
				},
			},
			LivenessProbe: container.LivenessProbe.convertToProbes(),
			StartupProbe:  container.StartupProbe.convertToProbes(),
		})
	}
	return containers
}

func (p *CloudRunProbe) convertToProbes() *runpb.Probe {
	if p == nil {
		return nil
	}
	probe := &runpb.Probe{
		InitialDelaySeconds: p.InitialDelaySeconds,
		PeriodSeconds:       p.PeriodSeconds,
		TimeoutSeconds:      p.TimeoutSeconds,
		FailureThreshold:    p.FailureThreshold,
	}
	switch p.ProbeSpec.ProbeType {
	case "HTTPGet":
		probe.ProbeType = &runpb.Probe_HttpGet{
			HttpGet: &runpb.HTTPGetAction{
				Path:        *p.ProbeSpec.Path,
				HttpHeaders: nil,
				Port:        p.ProbeSpec.Port,
			},
		}
	case "TCPSocket":
		probe.ProbeType = &runpb.Probe_TcpSocket{
			TcpSocket: &runpb.TCPSocketAction{
				Port: p.ProbeSpec.Port,
			},
		}
	case "Grpc":
		probe.ProbeType = &runpb.Probe_Grpc{
			Grpc: &runpb.GRPCAction{
				Port:    p.ProbeSpec.Port,
				Service: *p.ProbeSpec.Service,
			},
		}
	}
	return probe
}
