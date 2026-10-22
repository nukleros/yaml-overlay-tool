// Copyright 2021 VMware, Inc.
// SPDX-License-Identifier: MIT

// contains tests for the yaml-overlay-tool 'combine' action

package actions_test

import (
	"testing"

	"github.com/nukleros/yaml-overlay-tool/internal/actions"
)

const (
	queryBoolTestCase0 = "spec.boolTest.case0"
	queryBoolTestCase1 = "spec.boolTest.case1"
	valueTrue          = "true"
	valueFalse         = "false"
)

// expectedUnchangedBoolTest is the expected document for combine test cases
// whose combined boolTest value doesn't alter the rendered YAML.
const expectedUnchangedBoolTest = `apiVersion: v1
kind: Service
metadata:
  name: bind-udp
  namespace: nukleros-dns
  labels:
    app.kubernetes.io/name: external-dns
  annotations:
    # NOTE: this only works on 1.19.1+vmware.1+, but not prior
    ## This annotation will be ignored on other cloud providers
    service.beta.kubernetes.io/aws-load-balancer-type: nlb
spec:
  selector:
    app.kubernetes.io/name: external-dns
  type: LoadBalancer
  ports:
    - name: dns-udp
      port: 53
      protocol: UDP
      targetPort: dns-udp
    - name: dns-tcp
      port: 53
      protocol: TCP
      targetPort: dns-tcp
  # add some fake boolean values for testing
  boolTest:
    case0: false
    case1: true
`

func TestCombineNode(t *testing.T) {
	t.Parallel()

	tests := testCases{
		{
			name: "Combine Scalar Node - string",
			args: args{
				query: queryMetadataLabelsAppName,
				value: "es",
			},
			expectedValue: `apiVersion: v1
kind: Service
metadata:
  name: bind-udp
  namespace: nukleros-dns
  labels:
    app.kubernetes.io/name: external-dnses
  annotations:
    # NOTE: this only works on 1.19.1+vmware.1+, but not prior
    ## This annotation will be ignored on other cloud providers
    service.beta.kubernetes.io/aws-load-balancer-type: nlb
spec:
  selector:
    app.kubernetes.io/name: external-dns
  type: LoadBalancer
  ports:
    - name: dns-udp
      port: 53
      protocol: UDP
      targetPort: dns-udp
    - name: dns-tcp
      port: 53
      protocol: TCP
      targetPort: dns-tcp
  # add some fake boolean values for testing
  boolTest:
    case0: false
    case1: true
`,
		},
		{
			name: "Combine Scalar Node - integer",
			args: args{
				query: querySpecPortsSecondPort,
				value: "53",
			},
			expectedValue: `apiVersion: v1
kind: Service
metadata:
  name: bind-udp
  namespace: nukleros-dns
  labels:
    app.kubernetes.io/name: external-dns
  annotations:
    # NOTE: this only works on 1.19.1+vmware.1+, but not prior
    ## This annotation will be ignored on other cloud providers
    service.beta.kubernetes.io/aws-load-balancer-type: nlb
spec:
  selector:
    app.kubernetes.io/name: external-dns
  type: LoadBalancer
  ports:
    - name: dns-udp
      port: 53
      protocol: UDP
      targetPort: dns-udp
    - name: dns-tcp
      port: 106
      protocol: TCP
      targetPort: dns-tcp
  # add some fake boolean values for testing
  boolTest:
    case0: false
    case1: true
`,
		},
		{
			name: "Combine Scalar Node - true boolean with false boolean",
			args: args{
				query: queryBoolTestCase0,
				value: valueTrue,
			},
			expectedValue: expectedUnchangedBoolTest,
		},
		{
			name: "Combine Scalar Node - false boolean with false boolean",
			args: args{
				query: queryBoolTestCase0,
				value: valueFalse,
			},
			expectedValue: expectedUnchangedBoolTest,
		},
		{
			name: "Combine Scalar Node - true boolean with true boolean",
			args: args{
				query: queryBoolTestCase1,
				value: valueTrue,
			},
			expectedValue: expectedUnchangedBoolTest,
		},
		{
			name: "Combine Scalar Node - false boolean with true boolean",
			args: args{
				query: queryBoolTestCase1,
				value: valueFalse,
			},
			expectedValue: `apiVersion: v1
kind: Service
metadata:
  name: bind-udp
  namespace: nukleros-dns
  labels:
    app.kubernetes.io/name: external-dns
  annotations:
    # NOTE: this only works on 1.19.1+vmware.1+, but not prior
    ## This annotation will be ignored on other cloud providers
    service.beta.kubernetes.io/aws-load-balancer-type: nlb
spec:
  selector:
    app.kubernetes.io/name: external-dns
  type: LoadBalancer
  ports:
    - name: dns-udp
      port: 53
      protocol: UDP
      targetPort: dns-udp
    - name: dns-tcp
      port: 53
      protocol: TCP
      targetPort: dns-tcp
  # add some fake boolean values for testing
  boolTest:
    case0: false
    case1: false
`,
		},
		{
			name: "Combine Scalar Node - string (with comments)",
			args: args{
				query: "metadata.annotations['service.beta.kubernetes.io/aws-load-balancer-type']",
				value: "# head\ns #line\n# foot",
			},
			expectedValue: `apiVersion: v1
kind: Service
metadata:
  name: bind-udp
  namespace: nukleros-dns
  labels:
    app.kubernetes.io/name: external-dns
  annotations:
    # NOTE: this only works on 1.19.1+vmware.1+, but not prior
    ## This annotation will be ignored on other cloud providers
    service.beta.kubernetes.io/aws-load-balancer-type: nlbs #line
    # foot

    # head
spec:
  selector:
    app.kubernetes.io/name: external-dns
  type: LoadBalancer
  ports:
    - name: dns-udp
      port: 53
      protocol: UDP
      targetPort: dns-udp
    - name: dns-tcp
      port: 53
      protocol: TCP
      targetPort: dns-tcp
  # add some fake boolean values for testing
  boolTest:
    case0: false
    case1: true
`,
		},
	}

	tests.runTests(t, actions.CombineName)
}
