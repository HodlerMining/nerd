package svc_test

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/go-playground/validator"
	homedir "github.com/mitchellh/go-homedir"
	crd "github.com/nerdalize/nerd/crd/pkg/client/clientset/versioned"
	"github.com/nerdalize/nerd/pkg/kubevisor"
	"github.com/nerdalize/nerd/svc"
	"github.com/sirupsen/logrus"
)

func isNilErr(err error) bool {
	return err == nil
}

type testingDI struct {
	kube kubernetes.Interface
	crd  crd.Interface
	val  svc.Validator
	logs svc.Logger
	ns   string
}

func (di *testingDI) Kube() kubernetes.Interface {
	return di.kube
}

func (di *testingDI) Validator() svc.Validator {
	return di.val
}

func (di *testingDI) Logger() svc.Logger {
	return di.logs
}

func (di *testingDI) Namespace() string {
	return di.ns
}

func (di *testingDI) Crd() crd.Interface {
	return di.crd
}

func testNamespaceName(tb testing.TB) string {
	return fmt.Sprintf("%.63s", strings.ToLower(
		strings.Replace(
			strings.Replace(tb.Name(), "/", "-", -1), "_", "-", -1),
	))
}

func testDI(tb testing.TB) (svc.DI, func()) {
	tb.Helper()

	di, clean, err := svc.TempDI(testNamespaceName(tb))
	if err == svc.ErrMinikubeOnly {
		tb.Skipf("kube config needs to contain 'minikube' for local testing")
		return nil, nil
	}

	ok(tb, err)
	return di, clean
}

func testDIWithoutNamespace(tb testing.TB) svc.DI {
	tb.Helper()

	hdir, err := homedir.Dir()
	ok(tb, err)

	tdi := &testingDI{}
	kcfg, err := clientcmd.BuildConfigFromFlags("", filepath.Join(hdir, ".kube", "config"))
	ok(tb, err)

	if !strings.Contains(fmt.Sprintf("%#v", kcfg), "minikube") {
		tb.Skipf("kube config needs to contain 'minikube' for local testing")
	}

	tdi.logs = logrus.New()
	tdi.kube, err = kubernetes.NewForConfig(kcfg)
	ok(tb, err)

	tdi.val = validator.New()
	tdi.ns = "non-existing"
	return tdi
}

type testKube struct {
	visor *kubevisor.Visor
	val   svc.Validator
	logs  svc.Logger
}

func newTestKube(di svc.DI) (k *testKube) {
	k = &testKube{
		visor: kubevisor.NewVisor(di.Namespace(), "nlz-nerd", di.Kube(), di.Crd(), di.Logger()),
		val:   di.Validator(),
		logs:  di.Logger(),
	}

	return k
}

// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// ok fails the test if an err is not nil.
func ok(tb testing.TB, err error) {
	if err != nil {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d: unexpected error: %s\033[39m\n\n", filepath.Base(file), line, err.Error())
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%s:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}
