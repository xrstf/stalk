// SPDX-FileCopyrightText: 2023 Christoph Mewes
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"go.xrstf.de/stalk/pkg/diff"
	kubeutil "go.xrstf.de/stalk/pkg/kubernetes"
	"go.xrstf.de/stalk/pkg/watcher"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/watch"
	_ "k8s.io/client-go/plugin/pkg/client/auth/oidc"
	"k8s.io/client-go/tools/clientcmd"
)

// These variables get set by ldflags during compilation.
var (
	BuildTag    string
	BuildCommit string
	BuildDate   string // RFC3339 format ("2006-01-02T15:04:05Z07:00")
)

func printVersion() {
	// handle empty values in case `go install` was used
	if BuildCommit == "" {
		fmt.Printf("stalk dev, built with %s\n",
			runtime.Version(),
		)
	} else {
		fmt.Printf("stalk %s (%s), built with %s on %s\n",
			BuildTag,
			BuildCommit[:10],
			runtime.Version(),
			BuildDate,
		)
	}
}

type options struct {
	kubeconfig        string
	namespaces        []string
	labels            string
	hideManagedFields bool
	jsonPath          string
	hidePaths         []string
	showPaths         []string
	selector          labels.Selector
	showEmpty         bool
	disableWordDiff   bool
	contextLines      int
	verbose           bool
	version           bool
}

func main() {
	rootCtx := context.Background()

	opt := options{
		hideManagedFields: true,
		showEmpty:         false,
		disableWordDiff:   false,
		contextLines:      3,
	}

	pflag.StringVar(&opt.kubeconfig, "kubeconfig", opt.kubeconfig, "Kubeconfig file to use (uses $KUBECONFIG by default)")
	pflag.StringArrayVarP(&opt.namespaces, "namespace", "n", opt.namespaces, "Kubernetes namespace to watch resources in (supports glob expression) (can be given multiple times)")
	pflag.StringVarP(&opt.labels, "labels", "l", opt.labels, "Label-selector as an alternative to specifying resource names")
	pflag.BoolVar(&opt.hideManagedFields, "hide-managed", opt.hideManagedFields, "Do not show managed fields")
	pflag.StringVarP(&opt.jsonPath, "jsonpath", "j", opt.jsonPath, "JSON path expression to transform the output (applied before the --show paths)")
	pflag.StringArrayVarP(&opt.showPaths, "show", "s", opt.showPaths, "Path expression to include in output (can be given multiple times) (applied before the --hide paths)")
	pflag.StringArrayVarP(&opt.hidePaths, "hide", "h", opt.hidePaths, "Path expression to hide in output (can be given multiple times)")
	pflag.BoolVarP(&opt.showEmpty, "show-empty", "e", opt.showEmpty, "Do not hide changes which would produce no diff because of --hide/--show/--jsonpath")
	pflag.BoolVarP(&opt.disableWordDiff, "diff-by-line", "w", opt.disableWordDiff, "Compare entire lines and do not highlight changes within words")
	pflag.IntVarP(&opt.contextLines, "context-lines", "c", opt.contextLines, "Number of context lines to show in diffs")
	pflag.BoolVarP(&opt.verbose, "verbose", "v", opt.verbose, "Enable more verbose output")
	pflag.BoolVarP(&opt.version, "version", "V", opt.version, "Show version info and exit immediately")
	pflag.Parse()

	if opt.version {
		printVersion()
		return
	}

	// setup logging
	var log = logrus.New()
	log.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC1123,
	})

	if opt.verbose {
		log.SetLevel(logrus.DebugLevel)
	}

	// validate CLI flags
	differOpts := &diff.Options{
		ContextLines:     opt.contextLines,
		DisableWordDiff:  true,
		ExcludePaths:     opt.hidePaths,
		IncludePaths:     opt.showPaths,
		HideEmptyDiffs:   !opt.showEmpty,
		JSONPath:         opt.jsonPath,
		CreateColorTheme: diff.CreateColorTheme,
		UpdateColorTheme: diff.UpdateColorTheme,
		DeleteColorTheme: diff.DeleteColorTheme,
	}

	if opt.hideManagedFields {
		differOpts.ExcludePaths = append(differOpts.ExcludePaths, "metadata.managedFields")
	}

	if err := differOpts.Validate(); err != nil {
		log.Fatalf("Invalid CLI options: %v", err)
	}

	differ, err := diff.NewDiffer(differOpts, log)
	if err != nil {
		log.Fatalf("Failed to create differ: %v", err)
	}

	printer := diff.NewPrinter(differ, log)

	if opt.kubeconfig == "" {
		opt.kubeconfig = os.Getenv("KUBECONFIG")
	}

	args := pflag.Args()
	if len(args) == 0 {
		log.Fatal("No resource kind and name given.")
	}

	if args[0] == "-" {
		watchStdin(log, os.Stdin, printer)
	} else {
		watchKubernetes(rootCtx, log, args, &opt, printer)
	}
}

func watchStdin(log logrus.FieldLogger, input io.Reader, printer *diff.Printer) {
	decoder := yamlutil.NewYAMLOrJSONDecoder(input, 1024)

	for {
		object := unstructured.Unstructured{}
		err := decoder.Decode(&object)
		if err != nil {
			if err == io.EOF {
				break
			}

			log.Errorf("Failed to decode YAML object: %v", err)
			continue
		}

		printer.Print(&object, watch.Modified)
	}
}

func watchKubernetes(ctx context.Context, log logrus.FieldLogger, args []string, appOpts *options, printer *diff.Printer) {
	resourceKinds := strings.Split(strings.ToLower(args[0]), ",")
	resourceNames := args[1:]

	// is there a label selector?
	if appOpts.labels != "" {
		selector, err := labels.Parse(appOpts.labels)
		if err != nil {
			log.Fatalf("Invalid label selector: %v", err)
		}

		appOpts.selector = selector
	}

	hasNames := len(resourceNames) > 0
	if hasNames && appOpts.selector != nil {
		log.Fatal("Cannot specify both resource names and a label selector at the same time.")
	}

	// setup kubernetes client
	rules := clientcmd.NewDefaultClientConfigLoadingRules()
	rules.ExplicitPath = appOpts.kubeconfig

	deferred := clientcmd.NewInteractiveDeferredLoadingClientConfig(rules, &clientcmd.ConfigOverrides{}, os.Stdin)
	config, err := deferred.ClientConfig()
	if err != nil {
		log.Fatalf("Failed to create Kubernetes client: %v", err)
	}

	resolver, err := kubeutil.NewResolver(config, log)
	if err != nil {
		log.Fatalf("Failed to create Kubernetes REST mapper: %v", err)
	}

	// validate resource kinds
	log.Debug("Resolving resource kinds...")

	kinds := map[string]schema.GroupVersionKind{}
	for _, resourceKind := range resourceKinds {
		log.Debugf("Resolving %s...", resourceKind)

		parsed, err := resolver.Resolve(resourceKind)
		if err != nil {
			log.Fatalf("Unknown resource kind %q: %v", resourceKind, err)
		}

		//nolint:staticcheck
		if parsed == nil {
			log.Fatalf("Unknown resource kind %q", resourceKind)
		}

		//nolint:staticcheck
		gvk := parsed.GroupVersionKind
		kinds[gvk.String()] = gvk

		log.WithFields(logrus.Fields{
			"group":   gvk.Group,
			"version": gvk.Version,
			"kind":    gvk.Kind,
		}).Debug("Resolved")
	}

	// setup watches for each kind
	log.Debug("Starting to watch resources...")

	wg := sync.WaitGroup{}
	w := watcher.NewWatcher(printer, appOpts.namespaces, resourceNames)

	for _, gvk := range kinds {
		dynamicInterface, err := resolver.ResourceInterfaceFor(gvk)
		if err != nil {
			log.Fatalf("Failed to create dynamic interface for %q resources: %v", gvk.Kind, err)
		}

		wi, err := dynamicInterface.Watch(ctx, metav1.ListOptions{
			LabelSelector: appOpts.labels,
		})
		if err != nil {
			log.Fatalf("Failed to create watch for %q resources: %v", gvk.Kind, err)
		}

		wg.Add(1)
		go func() {
			w.Watch(ctx, wi)
			wg.Done()
		}()
	}

	wg.Wait()
}
