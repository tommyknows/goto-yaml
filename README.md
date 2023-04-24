# GoTo-YAML

This is a small utility written to transform Go variables & type declarations to
YAML files.

For example, when run against the `values` folder:

```bash
$ go run . ./values/

# DefaultValues defines the default values for the Helm chart

# Config defines the configuration for this Chart.
#
# this is a test comment
config:
    # X is cool
    x: hello
    # Y is not.
    #
    # we set Y to false because it's better that way.
    y: false
#
# y
image: hi
```

It started off because writing Helm chart values is tedious and annoying.
Writing the values in Go makes it possible to re-use already defined structs.
