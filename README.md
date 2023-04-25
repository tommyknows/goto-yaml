# GoTo-YAML

This is a small utility written to transform Go variables & type declarations to
YAML files.

For example, when run against the `values` folder:

```bash
$ go run . ./values/

# DefaultValues defines the default values for the Helm chart

# Number is an amount.
#
# Count defines the number of things.
#
# 8 is the best.
count: 8
# Config defines the configuration for an object.
#
# Config defines the configuration for this Chart.
#
# this is a test comment
config:
    # X is cool
    #
    # this is a test comment
    x: hello
    # Y is not.
    #
    # we set Y to false because it's better that way.
    y: false
    # map m defines greetings and goodbyes
    m:
        # map m defines greetings and goodbyes
        hello: world
        # sleep well little moon
        goodbye: moon
# y
image: hi
# y
other:
    # Truth checks whether this is true or not.
    #
    # We are not lying.
    truth: true
    # Values are cool.
    values:
        # does it?
        - hello
        # does this automagically work?
        - abc
```

It started off because writing Helm chart values is tedious and annoying.
Writing the values in Go makes it possible to re-use already defined structs.

## TODO

This project is in very early stages. It probably cannot handle many cases yet.
Some known areas of work:

- Testing: the example above works, anything else might break.
- It's very likely to panic in such cases.
