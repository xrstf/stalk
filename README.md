# Stalk - Watch your Kubernetes Resources change

`stalk` is a command line tool to watch a given set of resources and
print the diffs for every change.

## Installation

```bash
go get go.xrstf.de/stalk
```

## Usage

```
Usage of ./stalk:
  -h, --hide stringArray    path expression to hide in output (can be given multiple times)
      --hide-managed        Do not show managed fields (default true)
  -j, --jsonpath string     JSON path expression to transform the output (applied before the --show paths)
      --kubeconfig string   kubeconfig file to use (uses $KUBECONFIG by default)
  -l, --labels string       Label-selector as an alternative to specifying resource names
  -n, --namespace string    Kubernetes namespace to watch resources in (default "default")
  -s, --show stringArray    path expression to include in output (can be given multiple times) (applied before the --hide paths)
  -e, --show-empty          do not hide changes which would produce no diff because of --hide/--show/--jsonpath
  -v, --verbose             Enable more verbose output
```

## Examples

```bash
stalk -n kube-system deployments
```

Would watch all Deployments in the `kube-system` namespace.

```bash
stalk -n kube-system deployments,statefulsets,configmaps
```

Would also watch StatefulSets and ConfigMaps. Note that only a single
namespace can be given.

```bash
stalk -n kube-system deployments,statefulsets,configmaps,clusterroles
```

You can include Cluster-wide resources.

```bash
stalk -n kube-system deployments --selector "key=value"
```

A label selector can be given. It will be applied to all given resource kinds.

```bash
stalk -n kube-system deployments kube-apiserver kube-controller-manager kube-scheduler
```

You can also list the resources you are interested in by name.

```bash
stalk -n kube-system deployments --hide-managed-fields=false
```

By default `metadata.managedFields` is hidden. You can disable that if
you like.

```bash
stalk -n kube-system deployments --hide spec --hide metadata
```

Show only the `status`. You can combine `--hide` (`-h`) and `--show` (`-s`)
as you like, but show expressions are always applied before hide expressions.

```bash
stalk -n kube-system deployments --show spec --hide spec.labels
```

This should the entire spec, except the labels.

```bash
stalk -n kube-system deployments --jsonpath "{.metadata.name}"
```

JSONPaths are also supported, but only a single one can be given and it's always
applied first (before `--show` and `--hide`).

## License

MIT
