// Copyright Envoy Gateway Authors
// SPDX-License-Identifier: Apache-2.0
// The full text of the Apache license is available in the LICENSE file at
// the root of the repo.

package kubernetes

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	gwapiv1b1 "sigs.k8s.io/gateway-api/apis/v1beta1"

	"github.com/envoyproxy/gateway/api/config/v1alpha1"
	"github.com/envoyproxy/gateway/internal/envoygateway"
)

func TestIsAccepted(t *testing.T) {
	testCases := []struct {
		name   string
		gc     *gwapiv1b1.GatewayClass
		expect bool
	}{
		{
			name: "gatewayclass accepted condition",
			gc: &gwapiv1b1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: gwapiv1b1.GatewayClassSpec{
					ControllerName: gwapiv1b1.GatewayController(v1alpha1.GatewayControllerName),
				},
				Status: gwapiv1b1.GatewayClassStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(gwapiv1b1.GatewayClassConditionStatusAccepted),
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			expect: true,
		},
		{
			name: "gatewayclass not accepted condition",
			gc: &gwapiv1b1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: gwapiv1b1.GatewayClassSpec{
					ControllerName: gwapiv1b1.GatewayController(v1alpha1.GatewayControllerName),
				},
				Status: gwapiv1b1.GatewayClassStatus{
					Conditions: []metav1.Condition{
						{
							Type:   string(gwapiv1b1.GatewayClassConditionStatusAccepted),
							Status: metav1.ConditionFalse,
						},
					},
				},
			},
			expect: false,
		},
		{
			name: "no gatewayclass accepted condition type",
			gc: &gwapiv1b1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test",
				},
				Spec: gwapiv1b1.GatewayClassSpec{
					ControllerName: gwapiv1b1.GatewayController(v1alpha1.GatewayControllerName),
				},
				Status: gwapiv1b1.GatewayClassStatus{
					Conditions: []metav1.Condition{
						{
							Type:   "SomeOtherType",
							Status: metav1.ConditionTrue,
						},
					},
				},
			},
			expect: false,
		},
		{
			name:   "nil gatewayclass",
			expect: false,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			actual := isAccepted(tc.gc)
			require.Equal(t, tc.expect, actual)
		})
	}
}

func TestGatewaysOfClass(t *testing.T) {
	gc := &gwapiv1b1.GatewayClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test",
		},
	}
	testCases := []struct {
		name   string
		gws    []gwapiv1b1.Gateway
		expect int
	}{
		{
			name: "no matching gateways",
			gws: []gwapiv1b1.Gateway{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: gwapiv1b1.GatewaySpec{
						GatewayClassName: gwapiv1b1.ObjectName("no-match"),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: gwapiv1b1.GatewaySpec{
						GatewayClassName: gwapiv1b1.ObjectName("no-match2"),
					},
				},
			},
			expect: 0,
		},
		{
			name: "one of two matching gateways",
			gws: []gwapiv1b1.Gateway{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: gwapiv1b1.GatewaySpec{
						GatewayClassName: gwapiv1b1.ObjectName(gc.Name),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test2",
						Namespace: "test",
					},
					Spec: gwapiv1b1.GatewaySpec{
						GatewayClassName: gwapiv1b1.ObjectName("no-match"),
					},
				},
			},
			expect: 1,
		},
		{
			name: "two of two matching gateways",
			gws: []gwapiv1b1.Gateway{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test",
						Namespace: "test",
					},
					Spec: gwapiv1b1.GatewaySpec{
						GatewayClassName: gwapiv1b1.ObjectName(gc.Name),
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test2",
						Namespace: "test",
					},
					Spec: gwapiv1b1.GatewaySpec{
						GatewayClassName: gwapiv1b1.ObjectName(gc.Name),
					},
				},
			},
			expect: 2,
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			gwList := &gwapiv1b1.GatewayList{Items: tc.gws}
			actual := gatewaysOfClass(gc, gwList)
			require.Equal(t, tc.expect, len(actual))
		})
	}
}

func TestAddFinalizer(t *testing.T) {
	testCases := []struct {
		name   string
		gc     *gwapiv1b1.GatewayClass
		expect []string
	}{
		{
			name: "gatewayclass with no finalizers",
			gc: &gwapiv1b1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-gc",
				},
				Spec: gwapiv1b1.GatewayClassSpec{
					ControllerName: v1alpha1.GatewayControllerName,
				},
			},
			expect: []string{gatewayClassFinalizer},
		},
		{
			name: "gatewayclass with a different finalizer",
			gc: &gwapiv1b1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-gc",
					Finalizers: []string{"fooFinalizer"},
				},
				Spec: gwapiv1b1.GatewayClassSpec{
					ControllerName: v1alpha1.GatewayControllerName,
				},
			},
			expect: []string{"fooFinalizer", gatewayClassFinalizer},
		},
		{
			name: "gatewayclass with existing gatewayclass finalizer",
			gc: &gwapiv1b1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-gc",
					Finalizers: []string{gatewayClassFinalizer},
				},
				Spec: gwapiv1b1.GatewayClassSpec{
					ControllerName: v1alpha1.GatewayControllerName,
				},
			},
			expect: []string{gatewayClassFinalizer},
		},
	}

	// Create the reconciler.
	r := new(gatewayAPIReconciler)
	ctx := context.Background()

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			r.client = fakeclient.NewClientBuilder().WithScheme(envoygateway.GetScheme()).WithObjects(tc.gc).Build()
			err := r.addFinalizer(ctx, tc.gc)
			require.NoError(t, err)
			key := types.NamespacedName{Name: tc.gc.Name}
			err = r.client.Get(ctx, key, tc.gc)
			require.NoError(t, err)
			require.Equal(t, tc.expect, tc.gc.Finalizers)
		})
	}
}

func TestRemoveFinalizer(t *testing.T) {
	testCases := []struct {
		name   string
		gc     *gwapiv1b1.GatewayClass
		expect []string
	}{
		{
			name: "gatewayclass with no finalizers",
			gc: &gwapiv1b1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-gc",
				},
				Spec: gwapiv1b1.GatewayClassSpec{
					ControllerName: v1alpha1.GatewayControllerName,
				},
			},
			expect: nil,
		},
		{
			name: "gatewayclass with a different finalizer",
			gc: &gwapiv1b1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-gc",
					Finalizers: []string{"fooFinalizer"},
				},
				Spec: gwapiv1b1.GatewayClassSpec{
					ControllerName: v1alpha1.GatewayControllerName,
				},
			},
			expect: []string{"fooFinalizer"},
		},
		{
			name: "gatewayclass with existing gatewayclass finalizer",
			gc: &gwapiv1b1.GatewayClass{
				ObjectMeta: metav1.ObjectMeta{
					Name:       "test-gc",
					Finalizers: []string{gatewayClassFinalizer},
				},
				Spec: gwapiv1b1.GatewayClassSpec{
					ControllerName: v1alpha1.GatewayControllerName,
				},
			},
			expect: nil,
		},
	}

	// Create the reconciler.
	r := new(gatewayAPIReconciler)
	ctx := context.Background()

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			r.client = fakeclient.NewClientBuilder().WithScheme(envoygateway.GetScheme()).WithObjects(tc.gc).Build()
			err := r.removeFinalizer(ctx, tc.gc)
			require.NoError(t, err)
			key := types.NamespacedName{Name: tc.gc.Name}
			err = r.client.Get(ctx, key, tc.gc)
			require.NoError(t, err)
			require.Equal(t, tc.expect, tc.gc.Finalizers)
		})
	}
}
