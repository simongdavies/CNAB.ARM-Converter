package common

import (
	"context"
	"fmt"
	"io"
	"os"

	"get.porter.sh/porter/pkg/porter"
	"github.com/cnabio/cnab-go/bundle"
	"github.com/cnabio/cnab-to-oci/relocation"
	"github.com/cnabio/cnab-to-oci/remotes"
	"github.com/docker/cli/cli/config"
	"github.com/docker/distribution/reference"
)

var BuiltInActions = []string{
	"install",
	"upgrade",
	"uninstall",
}

type Options struct {
	OutputWriter          io.Writer
	Indent                bool
	Simplify              bool
	ReplaceKubeconfig     bool
	GenerateUI            bool
	IncludeCustomResource bool
	CustomRPTemplate      bool
	Debug                 bool
	Timeout               int
	UIWriter              io.Writer
	BundlePullOptions     *porter.BundlePullOptions
}

// BundleDetails is defines the bundle and bundle options to be used
type BundleDetails struct {
	BundleLoc string
	Options
}

func GetBundleDetails(options BundleDetails) (*bundle.Bundle, string, error) {
	useTag := false

	if options.BundlePullOptions.Tag != "" {
		useTag = true
	}

	bundle, err := getBundleFromTagOrFile(options.BundleLoc, useTag, options.BundlePullOptions)
	if err != nil {
		return nil, "", err
	}

	bundleTag := options.BundlePullOptions.Tag
	if !useTag {
		var err error
		bundleTag, err = getBundleTag(bundle)
		if err != nil {
			return nil, "", err
		}
	}
	return bundle, bundleTag, nil
}

func getBundleTag(bundle *bundle.Bundle) (string, error) {
	for _, i := range bundle.InvocationImages {
		if i.ImageType == "docker" {
			ref, err := reference.ParseNamed(i.Image)
			if err != nil {
				return "", fmt.Errorf("Cannot parse invocationImage reference: %s %w", i.Image, err)
			}

			bundleTag := ref.Name() + "/bundle"

			if tagged, ok := ref.(reference.Tagged); ok {
				bundleTag += ":"
				bundleTag += tagged.Tag()
			}

			if digested, ok := ref.(reference.Digested); ok {
				bundleTag += "@"
				bundleTag += digested.Digest().String()
			}

			return bundleTag, nil
		}
	}

	return "", fmt.Errorf("Cannot get bundle name from invocationImages: %v", bundle.InvocationImages)
}

func GetBundleFromTag(bundleOptions *porter.BundlePullOptions) (*bundle.Bundle, error) {
	// TODO deal with relocationMap
	bun, _, err := PullBundle(bundleOptions)
	if err != nil {
		return nil, fmt.Errorf("Unable to pull bundle with tag: %s. %w", bundleOptions.Tag, err)
	}
	return &bun, nil
}

func getBundleFromTagOrFile(source string, useTag bool, bundleOptions *porter.BundlePullOptions) (*bundle.Bundle, error) {
	if useTag {
		return GetBundleFromTag(bundleOptions)
	}

	return getBundleFromFile(source)
}

func getBundleFromFile(source string) (*bundle.Bundle, error) {
	_, err := os.Stat(source)
	if err != nil {
		return nil, fmt.Errorf("Unable to access bundle file: %s. %w", source, err)
	}
	jsonFile, _ := os.Open(source)
	bun, err := bundle.ParseReader(jsonFile)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse bundle file: %s. %w", source, err)
	}
	return &bun, nil
}

func PullBundle(bundlePullOptions *porter.BundlePullOptions) (bundle.Bundle, *relocation.ImageRelocationMap, error) {
	ref, err := reference.ParseNormalizedNamed(bundlePullOptions.Tag)
	if err != nil {
		return bundle.Bundle{}, nil, fmt.Errorf("Invalid bundle tag format %s, expected REGISTRY/name:tag %w", bundlePullOptions.Tag, err)
	}

	var insecureRegistries []string
	if bundlePullOptions.InsecureRegistry {
		reg := reference.Domain(ref)
		insecureRegistries = append(insecureRegistries, reg)
	}

	bun, reloMap, err := remotes.Pull(context.Background(), ref, remotes.CreateResolver(config.LoadDefaultConfigFile(os.Stderr), insecureRegistries...))
	if err != nil {
		return bundle.Bundle{}, nil, fmt.Errorf("Unable to pull remote bundle %w", err)
	}

	if len(reloMap) == 0 {
		return *bun, nil, nil
	}
	return *bun, &reloMap, nil
}
