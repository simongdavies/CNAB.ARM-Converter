module github.com/simongdavies/CNAB.ARM-Converter

go 1.14

replace (
	// https://github.com/cnabio/cnab-go/pull/229 (valueset-schema)
	github.com/cnabio/cnab-go => github.com/carolynvs/cnab-go v0.13.4-0.20200820201933-d6bf372247e5
	// See https://github.com/containerd/containerd/issues/3031
	// When I try to just use the require, go is shortening it to v2.7.1+incompatible which then fails to build...
	github.com/docker/distribution => github.com/docker/distribution v2.7.1-0.20190205005809-0d3efadf0154+incompatible
	github.com/docker/docker => github.com/moby/moby v0.7.3-0.20190826074503-38ab9da00309
	github.com/hashicorp/go-plugin => github.com/carolynvs/go-plugin v1.0.1-acceptstdin
)

require (
	get.porter.sh/porter v0.28.1
	github.com/cnabio/cnab-go v0.13.4-0.20200817181428-9005c1da4354
	github.com/cnabio/cnab-to-oci v0.3.1-beta1
	github.com/docker/cli v0.0.0-20191017083524-a8ff7f821017
	github.com/docker/distribution v2.7.1+incompatible
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.0.0
	gotest.tools v2.2.0+incompatible
)
