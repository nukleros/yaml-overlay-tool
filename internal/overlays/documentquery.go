// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package overlays

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"gopkg.in/yaml.v3"
)

type DocumentQuery struct {
	Conditions []*Condition `yaml:"conditions,omitempty"`
}

type Condition struct {
	Query Queries   `yaml:"query,omitempty"`
	Value yaml.Node `yaml:"value,omitempty"`
}

type DocumentQueries []*DocumentQuery

func (docQuery DocumentQueries) checkQueries(node *yaml.Node) bool {
	if len(docQuery) == 0 {
		log.Debugf("No Document Queries found, continuing")

		return true
	}

	for _, query := range docQuery {
		if ok := query.checkQuery(node); ok {
			log.Debugf("Document Query conditions were met, continuing")

			return true
		}
	}

	log.Debugf("Document Query conditions were not met, skipping")

	return false
}

func (docQuery *DocumentQuery) checkQuery(node *yaml.Node) bool {
	compareOptions := cmpopts.IgnoreFields(yaml.Node{}, "HeadComment", "LineComment", "FootComment", "Line", "Column", "Style")

	if docQuery.Conditions == nil {
		return true
	}

	for _, condition := range docQuery.Conditions {
		results := condition.Query.Find(node)
		if results == nil {
			return false
		}

		if !condition.Value.IsZero() {
			for _, result := range results {
				if ok := cmp.Equal(*result, condition.Value, compareOptions); !ok {
					return false
				}
			}
		}
	}

	return true
}
