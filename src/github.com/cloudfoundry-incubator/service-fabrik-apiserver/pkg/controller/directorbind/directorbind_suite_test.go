
//TODO copyright header


package directorbind_test

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	"github.com/kubernetes-incubator/apiserver-builder/pkg/test"

	"github.com/cloudfoundry-incubator/service-fabrik-apiserver/pkg/apis"
	"github.com/cloudfoundry-incubator/service-fabrik-apiserver/pkg/client/clientset_generated/clientset"
	"github.com/cloudfoundry-incubator/service-fabrik-apiserver/pkg/openapi"
	"github.com/cloudfoundry-incubator/service-fabrik-apiserver/pkg/controller/sharedinformers"
	"github.com/cloudfoundry-incubator/service-fabrik-apiserver/pkg/controller/directorbind"
)

var testenv *test.TestEnvironment
var config *rest.Config
var cs *clientset.Clientset
var shutdown chan struct{}
var controller *directorbind.DirectorBindController
var si *sharedinformers.SharedInformers

func TestDirectorBind(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecsWithDefaultAndCustomReporters(t, "DirectorBind Suite", []Reporter{test.NewlineReporter{}})
}

var _ = BeforeSuite(func() {
	testenv = test.NewTestEnvironment()
	config = testenv.Start(apis.GetAllApiBuilders(), openapi.GetOpenAPIDefinitions)
	cs = clientset.NewForConfigOrDie(config)

	shutdown = make(chan struct{})
	si = sharedinformers.NewSharedInformers(config, shutdown)
	controller = directorbind.NewDirectorBindController(config, si)
	controller.Run(shutdown)
})

var _ = AfterSuite(func() {
	close(shutdown)
	testenv.Stop()
})