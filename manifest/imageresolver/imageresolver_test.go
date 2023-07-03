package imageresolver_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	ociclient "github.com/fluxcd/pkg/oci/client"

	. "github.com/docker/labs-brown-tape/manifest/imageresolver"
	"github.com/docker/labs-brown-tape/manifest/imagescanner"
	"github.com/docker/labs-brown-tape/manifest/loader"
	"github.com/docker/labs-brown-tape/manifest/testdata"
)

func TestImageResover(t *testing.T) {
	cases := testdata.BaseYAMLCasesWithDigests(t)

	cases.Run(t, makeImageResolverTest)
}

func makeImageResolverTest(tc testdata.TestCase) func(t *testing.T) {
	return func(t *testing.T) {
		g := NewWithT(t)
		t.Parallel()

		loader := loader.NewRecursiveManifestDirectoryLoader(tc.Directory)
		g.Expect(loader.Load()).To(Succeed())

		expectedNumPaths := len(tc.Manifests)
		g.Expect(loader.Paths()).To(HaveLen(expectedNumPaths))

		scanner := imagescanner.NewDefaultImageScanner()

		g.Expect(scanner.Scan(loader.RelPaths())).To(Succeed())

		ctx := context.Background()
		client := ociclient.NewClient(nil)

		images := scanner.GetImages()

		// TODO: should this use fake resolver to avoid network traffic?
		g.Expect(NewRegistryResolver(client).ResolveDigests(ctx, images)).To(Succeed())

		if tc.Expected != nil {
			g.Expect(images.Items()).To(ConsistOf(tc.Expected))
		} else {
			t.Logf("%#v\n", images)
		}
	}
}
