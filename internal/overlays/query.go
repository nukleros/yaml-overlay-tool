// Copyright 2022 Nukleros
// Copyright 2021-2022 VMware, Inc.
// SPDX-License-Identifier: MIT

package overlays

import (
	"fmt"
	"strings"

	"github.com/nukleros/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
)

type Query struct {
	yamlPath    *yamlpath.Path
	queryString string
}

type Queries []Query

func (query *Query) UnmarshalYAML(value *yaml.Node) error {
	var queryString string

	if err := value.Decode(&queryString); err != nil {
		return fmt.Errorf("%w at line %d column %d", err, value.Line, value.Column)
	}

	yamlPath, err := yamlpath.NewPath(queryString)
	if err != nil {
		return fmt.Errorf("failed to parse the query path %s due to %w at line %d column %d", queryString, err, value.Line, value.Column)
	}

	query.yamlPath = yamlPath
	query.queryString = queryString

	return nil
}

func (query Query) String() string {
	return query.queryString
}

func (query *Query) Find(yamlNode *yaml.Node) []*yaml.Node {
	results, _ := query.yamlPath.Find(yamlNode)

	return results
}

func (multipleQueries *Queries) UnmarshalYAML(value *yaml.Node) error {
	var query Query

	if value.Kind == yaml.ScalarNode {
		if err := value.Decode(&query); err != nil {
			return fmt.Errorf("%w at line %d column %d", err, value.Line, value.Column)
		}

		*multipleQueries = []Query{query}

		return nil
	}

	type qualifiedQueries []Query

	if err := value.Decode((*qualifiedQueries)(multipleQueries)); err != nil {
		return fmt.Errorf("%w at line %d column %d", err, value.Line, value.Column)
	}

	return nil
}

func (multipleQueries *Queries) Paths() []string {
	queryPaths := make([]string, len(*multipleQueries))
	for i, query := range *multipleQueries {
		queryPaths[i] = query.String()
	}

	return queryPaths
}

func (multipleQueries Queries) String() string {
	return strings.Join(multipleQueries.Paths(), ",")
}

func (multipleQueries *Queries) Find(yamlNode *yaml.Node) []*yaml.Node {
	var results []*yaml.Node

	for _, query := range *multipleQueries {
		log.Debugf("searching path %s\n", query)

		if yamlNode.Kind == yaml.DocumentNode && len(yamlNode.Content) == 0 {
			continue
		}

		result := query.Find(yamlNode)

		results = append(results, result...)
	}

	return results
}

func (multipleQueries *Queries) Set(value string) error {
	var query Query
	if err := yaml.Unmarshal([]byte(value), &query); err != nil {
		return fmt.Errorf("%w", err)
	}

	*multipleQueries = append(*multipleQueries, query)

	return nil
}

func (mq *Queries) Type() string {
	return "overlays.Queries"
}
