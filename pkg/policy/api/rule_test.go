// SPDX-License-Identifier: Apache-2.0
// Copyright Authors of Cilium

//go:build !privileged_tests
// +build !privileged_tests

package api

import (
	"encoding/json"

	. "gopkg.in/check.v1"
)

func checkMarshalUnmarshal(c *C, r *Rule) {
	jsonData, err := json.Marshal(r)
	c.Assert(err, IsNil)

	newRule := Rule{}
	err = json.Unmarshal(jsonData, &newRule)
	c.Assert(err, IsNil)

	c.Check(newRule.EndpointSelector.LabelSelector == nil, Equals, r.EndpointSelector.LabelSelector == nil)
	c.Check(newRule.NodeSelector.LabelSelector == nil, Equals, r.NodeSelector.LabelSelector == nil)
}

// This test ensures that the NodeSelector and EndpointSelector fields are kept
// empty when the rule is marshalled/unmarshalled.
func (s *PolicyAPITestSuite) TestJSONMarshalling(c *C) {
	validEndpointRule := Rule{
		EndpointSelector: WildcardEndpointSelector,
	}
	checkMarshalUnmarshal(c, &validEndpointRule)

	validNodeRule := Rule{
		NodeSelector: WildcardEndpointSelector,
	}
	checkMarshalUnmarshal(c, &validNodeRule)
}
