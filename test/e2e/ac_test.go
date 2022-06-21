package e2e

import (
	"context"
	"time"

	edpCompApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	k8sClient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	adminConsoleApi "github.com/epam/edp-admin-console-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-admin-console-operator/v2/pkg/controller/helper"
	testHelper "github.com/epam/edp-admin-console-operator/v2/test/helper"
)

const (
	acCrName = "edp-admin-console"
)

var (
	env       *envtest.Environment
	client    k8sClient.Client
	namespace string
)

var _ = BeforeSuite(func() {
	env = &envtest.Environment{
		UseExistingCluster:       testHelper.GetBoolPointer(true),
		AttachControlPlaneOutput: true,
	}

	_, err := env.Start()
	Expect(err).ShouldNot(HaveOccurred())

	err = setupPrerequisite()
	Expect(err).ShouldNot(HaveOccurred())
})

func setupPrerequisite() error {
	var err error
	client, err = testHelper.InitClient(func(scheme *runtime.Scheme) {
		utilruntime.Must(clientgoscheme.AddToScheme(scheme))
		utilruntime.Must(adminConsoleApi.AddToScheme(scheme))
		utilruntime.Must(edpCompApi.AddToScheme(scheme))
	})
	if err != nil {
		return err
	}

	namespace, err = helper.GetWatchNamespace()
	if err != nil {
		return err
	}

	return nil
}

var _ = Describe("edp-admin-console integration", func() {
	Context("when sso is disabled", func() {
		Context("when db is disabled", func() {
			Context("when admin-console cr is created by helm (default deploying flow)", func() {
				It("when all fields are valid", func() {
					err := retry(15, 10*time.Second, func() (*adminConsoleApi.AdminConsole, error) {
						ac := &adminConsoleApi.AdminConsole{}
						err := client.Get(context.TODO(), types.NamespacedName{
							Name:      acCrName,
							Namespace: namespace,
						}, ac)
						if err != nil {
							return nil, err
						}
						return ac, nil
					})
					Expect(err).ShouldNot(HaveOccurred())

					ec := &edpCompApi.EDPComponent{}
					err = client.Get(context.TODO(), types.NamespacedName{
						Name:      acCrName,
						Namespace: namespace,
					}, ec)
					Expect(err).ShouldNot(HaveOccurred())
				})
			})
			Context("when admin-console cr is created by tests", func() {
				It("when all fields are valid", func() {
					ac := &adminConsoleApi.AdminConsole{
						ObjectMeta: metav1.ObjectMeta{
							Name:      acCrName,
							Namespace: namespace,
						},
						Spec: adminConsoleApi.AdminConsoleSpec{
							KeycloakSpec: adminConsoleApi.KeycloakSpec{
								Enabled: false,
							},
							EdpSpec: adminConsoleApi.EdpSpec{
								IntegrationStrategies: "Create,Clone,Import",
								TestReportTools:       "Allure",
							},
							DbSpec: adminConsoleApi.AdminConsoleDbSettings{
								Enabled: true,
							},
							BasePath: "",
						},
					}
					err := client.Create(context.TODO(), ac)
					Expect(err).ShouldNot(HaveOccurred())

					err = retry(15, 10*time.Second, func() (*adminConsoleApi.AdminConsole, error) {
						ac := &adminConsoleApi.AdminConsole{}
						err := client.Get(context.TODO(), types.NamespacedName{
							Name:      acCrName,
							Namespace: namespace,
						}, ac)
						if err != nil {
							return nil, err
						}
						return ac, nil
					})
					Expect(err).ShouldNot(HaveOccurred())

					ec := &edpCompApi.EDPComponent{}
					err = client.Get(context.TODO(), types.NamespacedName{
						Name:      acCrName,
						Namespace: namespace,
					}, ec)
					Expect(err).ShouldNot(HaveOccurred())
				})
			})
		})
		BeforeEach(func() {
			time.Sleep(5 * time.Second)
		})
		// cleanup ac cr
		AfterEach(func() {
			err := client.Delete(context.TODO(), &adminConsoleApi.AdminConsole{
				ObjectMeta: metav1.ObjectMeta{
					Name:      acCrName,
					Namespace: namespace,
				},
			})
			Expect(err).ShouldNot(HaveOccurred())
		})
	})
})

func retry(attempts int, sleep time.Duration, f func() (*adminConsoleApi.AdminConsole, error)) error {
	for i := 0; ; i++ {
		ac, err := f()
		if err != nil {
			return err
		}

		if ac.Status.Available {
			return nil
		}

		if i >= (attempts - 1) {
			return errors.New("admin console status failed")
		}

		time.Sleep(sleep)
	}
}
