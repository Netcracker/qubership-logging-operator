This directory holds the configuration for generating the Markdown reference
documentation of the operator's custom resource definitions. It uses the
[crd-ref-docs](https://github.com/elastic/crd-ref-docs) project.

`config.yaml` lists the types and fields to leave out of the output and pins the
Kubernetes version used for links to the upstream API reference. The renderer
uses the templates built into `crd-ref-docs`.

## Building

From the project's top directory, run:

```console
make docs/api.md
```
