// SPDX-License-Identifier: Apache-2.0
// Copyright 2021 Authors of Cilium

//go:build !privileged_tests
// +build !privileged_tests

package flows_to_world

import (
	"context"
	"strings"
	"testing"

	flowpb "github.com/cilium/cilium/api/v1/flow"
	"github.com/cilium/cilium/pkg/hubble/metrics/api"
	monitorAPI "github.com/cilium/cilium/pkg/monitor/api"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestFlowsToWorldHandler_MatchingFlow(t *testing.T) {
	registry := prometheus.NewRegistry()
	opts := api.Options{"sourceContext": "namespace", "destinationContext": "dns|ip"}
	h := &flowsToWorldHandler{}
	assert.NoError(t, h.Init(registry, opts))
	assert.NoError(t, testutil.CollectAndCompare(h.flowsToWorld, strings.NewReader("")))
	flow := flowpb.Flow{
		Verdict:        flowpb.Verdict_DROPPED,
		DropReasonDesc: flowpb.DropReason_POLICY_DENIED,
		EventType:      &flowpb.CiliumEventType{Type: monitorAPI.MessageTypeDrop},
		L4: &flowpb.Layer4{
			Protocol: &flowpb.Layer4_TCP{
				TCP: &flowpb.TCP{DestinationPort: 80},
			},
		},
		Source: &flowpb.Endpoint{Namespace: "src-a"},
		Destination: &flowpb.Endpoint{
			Labels: []string{"reserved:world"},
		},
		DestinationNames: []string{"cilium.io"},
	}

	h.ProcessFlow(context.Background(), &flow)
	flow.L4 = &flowpb.Layer4{
		Protocol: &flowpb.Layer4_UDP{UDP: &flowpb.UDP{DestinationPort: 53}},
	}
	h.ProcessFlow(context.Background(), &flow)
	flow.L4 = &flowpb.Layer4{
		Protocol: &flowpb.Layer4_ICMPv4{ICMPv4: &flowpb.ICMPv4{}},
	}
	h.ProcessFlow(context.Background(), &flow)
	flow.L4 = &flowpb.Layer4{
		Protocol: &flowpb.Layer4_ICMPv6{ICMPv6: &flowpb.ICMPv6{}},
	}
	h.ProcessFlow(context.Background(), &flow)
	expected := strings.NewReader(`# HELP hubble_flows_to_world_total Total number of flows to reserved:world
# TYPE hubble_flows_to_world_total counter
hubble_flows_to_world_total{destination="cilium.io",protocol="ICMPv4",source="src-a",verdict="DROPPED"} 1
hubble_flows_to_world_total{destination="cilium.io",protocol="ICMPv6",source="src-a",verdict="DROPPED"} 1
hubble_flows_to_world_total{destination="cilium.io",protocol="UDP",source="src-a",verdict="DROPPED"} 1
hubble_flows_to_world_total{destination="cilium.io",protocol="TCP",source="src-a",verdict="DROPPED"} 1
`)
	assert.NoError(t, testutil.CollectAndCompare(h.flowsToWorld, expected))
}

func TestFlowsToWorldHandler_NonMatchingFlows(t *testing.T) {
	registry := prometheus.NewRegistry()
	opts := api.Options{"sourceContext": "namespace", "destinationContext": "dns|ip"}
	h := &flowsToWorldHandler{}
	assert.NoError(t, h.Init(registry, opts))

	// destination is missing.
	h.ProcessFlow(context.Background(), &flowpb.Flow{
		Verdict: flowpb.Verdict_FORWARDED,
		Source:  &flowpb.Endpoint{Namespace: "src-a"},
	})
	// destination is not reserved:world
	h.ProcessFlow(context.Background(), &flowpb.Flow{
		Verdict: flowpb.Verdict_FORWARDED,
		Source:  &flowpb.Endpoint{Namespace: "src-a"},
		Destination: &flowpb.Endpoint{
			Labels: []string{"reserved:host"},
		},
	})
	// L4 information is missing.
	h.ProcessFlow(context.Background(), &flowpb.Flow{
		Verdict: flowpb.Verdict_FORWARDED,
		Source:  &flowpb.Endpoint{Namespace: "src-a"},
		Destination: &flowpb.Endpoint{
			Labels: []string{"reserved:world"},
		},
	})
	// EventType is missing.
	h.ProcessFlow(context.Background(), &flowpb.Flow{
		Verdict: flowpb.Verdict_FORWARDED,
		Source:  &flowpb.Endpoint{Namespace: "src-a"},
		Destination: &flowpb.Endpoint{
			Labels: []string{"reserved:world"},
		},
		L4: &flowpb.Layer4{
			Protocol: &flowpb.Layer4_TCP{
				TCP: &flowpb.TCP{DestinationPort: 80},
			},
		},
	})
	// Drop reason is not "Policy denied".
	h.ProcessFlow(context.Background(), &flowpb.Flow{
		Verdict:        flowpb.Verdict_DROPPED,
		EventType:      &flowpb.CiliumEventType{Type: monitorAPI.MessageTypeDrop},
		DropReasonDesc: flowpb.DropReason_STALE_OR_UNROUTABLE_IP,
		L4: &flowpb.Layer4{
			Protocol: &flowpb.Layer4_TCP{
				TCP: &flowpb.TCP{DestinationPort: 80},
			},
		},
		Source: &flowpb.Endpoint{Namespace: "src-a"},
		Destination: &flowpb.Endpoint{
			Labels: []string{"reserved:world"},
		},
		DestinationNames: []string{"cilium.io"},
	})
	// Flow is a reply.
	h.ProcessFlow(context.Background(), &flowpb.Flow{
		Verdict:   flowpb.Verdict_FORWARDED,
		EventType: &flowpb.CiliumEventType{Type: monitorAPI.MessageTypeTrace},
		L4: &flowpb.Layer4{
			Protocol: &flowpb.Layer4_TCP{
				TCP: &flowpb.TCP{DestinationPort: 80},
			},
		},
		Source: &flowpb.Endpoint{Namespace: "src-a"},
		Destination: &flowpb.Endpoint{
			Labels: []string{"reserved:world"},
		},
		DestinationNames: []string{"cilium.io"},
		IsReply:          wrapperspb.Bool(true),
	})
	assert.NoError(t, testutil.CollectAndCompare(h.flowsToWorld, strings.NewReader("")))
}

func TestFlowsToWorldHandler_AnyDrop(t *testing.T) {
	registry := prometheus.NewRegistry()
	opts := api.Options{"sourceContext": "namespace", "destinationContext": "dns|ip", "any-drop": ""}
	h := &flowsToWorldHandler{}
	assert.NoError(t, h.Init(registry, opts))
	assert.NoError(t, testutil.CollectAndCompare(h.flowsToWorld, strings.NewReader("")))
	flow := flowpb.Flow{
		Verdict:        flowpb.Verdict_DROPPED,
		DropReasonDesc: flowpb.DropReason_STALE_OR_UNROUTABLE_IP,
		EventType:      &flowpb.CiliumEventType{Type: monitorAPI.MessageTypeDrop},
		L4: &flowpb.Layer4{
			Protocol: &flowpb.Layer4_TCP{
				TCP: &flowpb.TCP{DestinationPort: 80},
			},
		},
		Source: &flowpb.Endpoint{Namespace: "src-a"},
		Destination: &flowpb.Endpoint{
			Labels: []string{"reserved:world"},
		},
		DestinationNames: []string{"cilium.io"},
	}
	h.ProcessFlow(context.Background(), &flow)
	expected := strings.NewReader(`# HELP hubble_flows_to_world_total Total number of flows to reserved:world
# TYPE hubble_flows_to_world_total counter
hubble_flows_to_world_total{destination="cilium.io",protocol="TCP",source="src-a",verdict="DROPPED"} 1
`)
	assert.NoError(t, testutil.CollectAndCompare(h.flowsToWorld, expected))
}

func TestFlowsToWorldHandler_IncludePort(t *testing.T) {
	registry := prometheus.NewRegistry()
	opts := api.Options{"sourceContext": "namespace", "destinationContext": "dns|ip", "port": ""}
	h := &flowsToWorldHandler{}
	assert.NoError(t, h.Init(registry, opts))
	assert.NoError(t, testutil.CollectAndCompare(h.flowsToWorld, strings.NewReader("")))
	flow := flowpb.Flow{
		Verdict:   flowpb.Verdict_FORWARDED,
		EventType: &flowpb.CiliumEventType{Type: monitorAPI.MessageTypeTrace},
		L4: &flowpb.Layer4{
			Protocol: &flowpb.Layer4_TCP{
				TCP: &flowpb.TCP{DestinationPort: 80},
			},
		},
		Source: &flowpb.Endpoint{Namespace: "src-a"},
		Destination: &flowpb.Endpoint{
			Labels: []string{"reserved:world"},
		},
		DestinationNames: []string{"cilium.io"},
		IsReply:          wrapperspb.Bool(false),
	}
	h.ProcessFlow(context.Background(), &flow)
	expected := strings.NewReader(`# HELP hubble_flows_to_world_total Total number of flows to reserved:world
# TYPE hubble_flows_to_world_total counter
hubble_flows_to_world_total{destination="cilium.io",port="80",protocol="TCP",source="src-a",verdict="FORWARDED"} 1
`)
	assert.NoError(t, testutil.CollectAndCompare(h.flowsToWorld, expected))
}
