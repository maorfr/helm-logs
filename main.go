package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/golang/protobuf/proto"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	rspb "k8s.io/helm/pkg/proto/hapi/release"
)

var (
	namespace       string
	storage         string
	since           time.Duration
	tillerNamespace string
	label           string
)

func main() {
	cmd := &cobra.Command{
		Use:   "logs [flags]",
		Short: "",
		RunE:  run,
	}

	f := cmd.Flags()
	f.StringVar(&namespace, "namespace", "", "show releases within a specific namespace")
	f.StringVar(&storage, "storage", "cfgmaps", "storage type of releases. One of: 'cfgmaps', 'secrets'")
	f.DurationVar(&since, "since", time.Duration(1000000*time.Hour), "Only return logs newer than a relative duration like 5s, 2m, or 3h. Defaults to all logs.")
	f.StringVar(&tillerNamespace, "tiller-namespace", "kube-system", "namespace of Tiller")
	f.StringVarP(&label, "label", "l", "OWNER=TILLER", "label to select tiller resources by")

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	releases, printData, err := listReleases(namespace, storage, tillerNamespace, label, since)
	if err != nil {
		return err
	}

	print(releases, *printData)
	return nil
}

type releaseData struct {
	name      string
	revision  int32
	updated   string
	status    string
	chart     string
	namespace string
	time      time.Time
}

type printColumnWidthData struct {
	name      int
	revision  int
	updated   int
	status    int
	chart     int
	namespace int
}

func listReleases(namespace, storage, tillerNamespace, label string, since time.Duration) ([]releaseData, *printColumnWidthData, error) {
	k8sClient, err := GetClientToK8s()
	if err != nil {
		return nil, nil, err
	}
	var releasesData []releaseData
	printData := initPrintData()
	coreV1 := k8sClient.ClientSet.CoreV1()
	switch storage {
	case "secrets":
		secrets, err := coreV1.Secrets(tillerNamespace).List(metav1.ListOptions{
			LabelSelector: label,
		})
		if err != nil {
			return nil, nil, err
		}
		for _, item := range secrets.Items {
			releaseData := getReleaseData((string)(item.Data["release"]))
			if releaseData == nil {
				continue
			}
			printData = getPrintData(*releaseData, printData)
			releasesData = append(releasesData, *releaseData)
		}
	case "cfgmaps":
		configMaps, err := coreV1.ConfigMaps(tillerNamespace).List(metav1.ListOptions{
			LabelSelector: label,
		})
		if err != nil {
			return nil, nil, err
		}
		for _, item := range configMaps.Items {
			releaseData := getReleaseData(item.Data["release"])
			if releaseData == nil {
				continue
			}
			printData = getPrintData(*releaseData, printData)
			releasesData = append(releasesData, *releaseData)
		}
	}

	sort.Slice(releasesData[:], func(i, j int) bool {
		return releasesData[i].time.Before(releasesData[j].time)
	})

	return releasesData, &printData, nil
}

func getReleaseData(itemReleaseData string) *releaseData {
	data, _ := decodeRelease(itemReleaseData)

	if namespace != "" && data.Namespace != namespace {
		return nil
	}
	deployTime := time.Unix(data.Info.LastDeployed.Seconds, 0)
	if deployTime.Before(time.Now().Add(-since)) {
		return nil
	}
	chartMeta := data.GetChart().Metadata
	releaseData := releaseData{
		time:      deployTime,
		name:      data.Name,
		revision:  data.Version,
		updated:   deployTime.Format("Mon Jan _2 15:04:05 2006"),
		status:    data.GetInfo().Status.Code.String(),
		chart:     fmt.Sprintf("%s-%s", chartMeta.Name, chartMeta.Version),
		namespace: data.Namespace,
	}
	return &releaseData
}

func initPrintData() printColumnWidthData {
	return printColumnWidthData{
		name:      len("NAME"),
		revision:  len("REVISION"),
		updated:   len("Mon Jan _2 15:04:05 2006"),
		status:    len("STATUS"),
		chart:     len("CHART"),
		namespace: len("NAMESPACE"),
	}
}

func getPrintData(releaseData releaseData, printData printColumnWidthData) printColumnWidthData {
	printData.name = int(math.Max(float64(printData.name), float64(len(releaseData.name))))
	printData.status = int(math.Max(float64(printData.status), float64(len(releaseData.status))))
	printData.chart = int(math.Max(float64(printData.chart), float64(len(releaseData.chart))))
	printData.namespace = int(math.Max(float64(printData.namespace), float64(len(releaseData.namespace))))
	return printData
}

// K8sClient holds a clientset and a config
type K8sClient struct {
	ClientSet *kubernetes.Clientset
	Config    *rest.Config
}

// GetClientToK8s returns a k8sClient
func GetClientToK8s() (*K8sClient, error) {
	var kubeconfig string
	if kubeConfigPath := os.Getenv("KUBECONFIG"); kubeConfigPath != "" {
		kubeconfig = kubeConfigPath // CI process
	} else {
		kubeconfig = filepath.Join(os.Getenv("HOME"), ".kube", "config") // Development environment
	}

	var config *rest.Config

	_, err := os.Stat(kubeconfig)
	if err != nil {
		// In cluster configuration
		config, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	} else {
		// Out of cluster configuration
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	var client = &K8sClient{ClientSet: clientset, Config: config}
	return client, nil
}

var b64 = base64.StdEncoding
var magicGzip = []byte{0x1f, 0x8b, 0x08}

func decodeRelease(data string) (*rspb.Release, error) {
	// base64 decode string
	b, err := b64.DecodeString(data)
	if err != nil {
		return nil, err
	}

	// For backwards compatibility with releases that were stored before
	// compression was introduced we skip decompression if the
	// gzip magic header is not found
	if bytes.Equal(b[0:3], magicGzip) {
		r, err := gzip.NewReader(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		b2, err := ioutil.ReadAll(r)
		if err != nil {
			return nil, err
		}
		b = b2
	}

	var rls rspb.Release
	// unmarshal protobuf bytes
	if err := proto.Unmarshal(b, &rls); err != nil {
		return nil, err
	}
	return &rls, nil
}

func print(releases []releaseData, printData printColumnWidthData) {
	if len(releases) == 0 {
		return
	}

	nameCW := strconv.Itoa(printData.name)
	revisionCW := strconv.Itoa(printData.revision)
	updatedCW := strconv.Itoa(printData.updated)
	statusCW := strconv.Itoa(printData.status)
	chartCW := strconv.Itoa(printData.chart)
	namespaceCW := strconv.Itoa(printData.namespace)

	fmt.Printf("%-"+nameCW+"s\t%-"+revisionCW+"s\t%-"+updatedCW+"s\t%-"+statusCW+"s\t%-"+chartCW+"s\t%-"+namespaceCW+"s\n", "NAME", "REVISION", "UPDATED", "STATUS", "CHART", "NAMESPACE")
	for _, r := range releases {
		fmt.Printf("%-"+nameCW+"s\t%-"+revisionCW+"d\t%-"+updatedCW+"s\t%-"+statusCW+"s\t%-"+chartCW+"s\t%-"+namespaceCW+"s\n", r.name, r.revision, r.updated, r.status, r.chart, r.namespace)
	}
}
