package controllers

import (
	"context"
	"github.com/SAP/sap-btp-service-operator/api/v1alpha1"
	"github.com/SAP/sap-btp-service-operator/internal/secrets"
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

const managementNamespace = "test-management-namespace"

var _ = Describe("Base controller", func() {
	var serviceInstance *v1alpha1.ServiceInstance
	var fakeInstanceName string
	var ctx context.Context
	var controller *BaseReconciler

	BeforeEach(func() {
		ctx = context.Background()
		fakeInstanceName = "ic-test-" + uuid.New().String()

		resolver := &secrets.SecretResolver{
			ManagementNamespace: managementNamespace,
			Log:                 logf.Log.WithName("SecretResolver"),
			Client:              k8sClient,
		}
		controller = &BaseReconciler{
			SecretResolver: resolver,
			Log:            logf.Log.WithName("reconciler"),
			Client:         k8sClient,
		}
		serviceInstance = &v1alpha1.ServiceInstance{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "services.cloud.sap.com/v1alpha1",
				Kind:       "ServiceInstance",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      fakeInstanceName,
				Namespace: testNamespace,
			},
			Spec: v1alpha1.ServiceInstanceSpec{
				ExternalName:        fakeInstanceExternalName,
				ServicePlanName:     fakePlanName,
				ServiceOfferingName: fakeOfferingName,
			},
		}
	})

	When("SM secret not exists", func() {
		It("Should fail with failure condition", func() {
			controller.getSMClient(ctx, controller.Log, serviceInstance)
			Expect(serviceInstance.Status.Conditions[0].Reason).To(Equal(Blocked))
			Expect(len(serviceInstance.Status.Conditions)).To(Equal(1))
		})
	})

	When("SM secret is valid", func() {
		var namespace *corev1.Namespace
		var secret *corev1.Secret
		BeforeEach(func() {
			namespace = &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: managementNamespace}}
			Expect(k8sClient.Create(context.Background(), namespace)).Should(Succeed())
			secret = &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      secrets.SAPBTPOperatorSecretName,
					Namespace: managementNamespace,
				},
				Data: map[string][]byte{
					"clientid":     []byte("client-id"),
					"clientsecret": []byte("client-secret"),
					"url":          []byte("https://some.url"),
					"tokenurl":     []byte("https://token.url"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).Should(Succeed())
		})
		AfterEach(func() {
			Expect(k8sClient.Delete(ctx, secret)).Should(Succeed())
			Eventually(func() bool {
				err := k8sClient.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, secret)
				return apierrors.IsNotFound(err)
			}, timeout, interval).Should(BeTrue())
			Expect(k8sClient.Delete(ctx, namespace)).Should(Succeed())
		})
		It("Should succeed", func() {
			client, err := controller.getSMClient(ctx, controller.Log, serviceInstance)
			Expect(err).To(BeNil())
			Expect(client).ToNot(BeNil())
		})
	})
})
