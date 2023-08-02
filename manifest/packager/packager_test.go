package packager_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/google/uuid"
	. "github.com/onsi/gomega"

	"github.com/docker/labs-brown-tape/manifest/imagecopier"
	"github.com/docker/labs-brown-tape/manifest/imageresolver"
	"github.com/docker/labs-brown-tape/manifest/imagescanner"
	"github.com/docker/labs-brown-tape/manifest/loader"
	. "github.com/docker/labs-brown-tape/manifest/packager"
	"github.com/docker/labs-brown-tape/manifest/testdata"
	"github.com/docker/labs-brown-tape/manifest/updater"
	"github.com/docker/labs-brown-tape/oci"
)

var destinationUUID = uuid.New().String()

func makeDestination(name string) string {
	return fmt.Sprintf("ttl.sh/%s/bpt-packager-test-%s", destinationUUID, name)
}

func TestUpdater(t *testing.T) {
	cases := testdata.BaseYAMLCasesWithDigests(t)
	cases.Run(t, makeUpdaterTest)
}

func makeUpdaterTest(tc testdata.TestCase) func(t *testing.T) {
	return func(t *testing.T) {
		g := NewWithT(t)
		t.Parallel()

		loader := loader.NewRecursiveManifestDirectoryLoader(tc.Directory)
		g.Expect(loader.Load()).To(Succeed())

		scanner := imagescanner.NewDefaultImageScanner()

		expectedNumPaths := len(tc.Manifests)
		g.Expect(loader.Paths()).To(HaveLen(expectedNumPaths))

		for i := range tc.Manifests {
			g.Expect(loader.ContainsRelPath(tc.Manifests[i])).To(BeTrue())
		}

		g.Expect(scanner.Scan(loader.RelPaths())).To(Succeed())

		ctx := context.Background()
		client := oci.NewClient(nil)

		images := scanner.GetImages()

		// TODO: should this use fake resolver to avoid network traffic?
		g.Expect(imageresolver.NewRegistryResolver(client).ResolveDigests(ctx, images)).To(Succeed())

		imagesCopied, err := imagecopier.NewRegistryCopier(client, makeDestination(tc.Description)).CopyImages(ctx, images)
		g.Expect(err).ToNot(HaveOccurred())
		g.Expect(imagesCopied).To(HaveLen(images.Len()))

		imagecopier.SetNewImageRefs(makeDestination(tc.Description), sha256.New(), tc.Expected)

		g.Expect(updater.NewFileUpdater().Update(images)).To(Succeed())

		if tc.Expected != nil {
			g.Expect(images.Items()).To(ConsistOf(tc.Expected))
		} else {
			t.Logf("%#v\n", images)
		}

		// scanner.Reset()

		// g.Expect(scanner.Scan(loader.RelPaths())).To(Succeed())

		_, err = NewDefaultPackager(client, makeDestination(tc.Description)).Push(ctx, images.Dir())
		g.Expect(err).To(Succeed())

		// TODO: pull the contents from the registry and compare them to what is expected;
		// e.g. also as the means to test inspection logic (TBI)
	}
}
