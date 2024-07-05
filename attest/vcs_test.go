package attest_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/google/go-containerregistry/pkg/name"

	. "github.com/onsi/gomega"

	. "github.com/errordeveloper/tape/attest"
	"github.com/errordeveloper/tape/attest/vcs/git"
	"github.com/errordeveloper/tape/manifest/imagescanner"
	"github.com/errordeveloper/tape/manifest/loader"
	"github.com/errordeveloper/tape/oci"
)

type vcsTestCase struct {
	URL, CheckoutTag, CheckoutHash, Branch          string
	LoadPath                                        string
	ExpectManifests, ExpectImageTags, ExpectRawTags []string
}

func (tc vcsTestCase) Name() string {
	rev := tc.CheckoutTag
	if rev == "" {
		rev = tc.CheckoutHash
	}
	return fmt.Sprintf("%s@%s", tc.URL, rev)
}

func TestVCS(t *testing.T) {
	testCases := []vcsTestCase{
		// {
		// 	URL:             "https://github.com/stefanprodan/podinfo",
		// 	CheckoutTag:     "6.7.0", // => 0b1481aa8ed0a6c34af84f779824a74200d5c1d6
		// 	LoadPath:        "kustomize",
		// 	ExpectManifests: []string{"kustomization.yaml", "deployment.yaml", "hpa.yaml", "service.yaml"},
		// 	ExpectImageTags: []string{"6.7.0"},
		// 	ExpectRawTags:   []string{"6.7.0"},
		// },
		// {
		// 	URL:             "https://github.com/stefanprodan/podinfo",
		// 	CheckoutHash:    "0b1481aa8ed0a6c34af84f779824a74200d5c1d6", // => 6.7.0
		// 	Branch:          "master",
		// 	LoadPath:        "kustomize",
		// 	ExpectManifests: []string{"kustomization.yaml", "deployment.yaml", "hpa.yaml", "service.yaml"},
		// 	ExpectImageTags: []string{"6.7.0"},
		// 	ExpectRawTags:   []string{"6.7.0"},
		// },
		{
			URL:             "https://github.com/errordeveloper/tape-git-testing",
			CheckoutHash:    "3cad1d255c1d83b5e523de64d34758609498d81b",
			Branch:          "main",
			LoadPath:        "",
			ExpectManifests: []string{"kustomization.yaml", "deployment.yaml", "hpa.yaml", "service.yaml"},
			ExpectImageTags: nil,
			ExpectRawTags:   nil,
		},
		{
			URL:             "https://github.com/errordeveloper/tape-git-testing",
			CheckoutTag:     "0.0.1",
			LoadPath:        "",
			ExpectManifests: []string{"podinfo/kustomization.yaml", "podinfo/deployment.yaml", "podinfo/hpa.yaml", "podinfo/service.yaml"},
			ExpectImageTags: []string{"v0.0.1"},
			ExpectRawTags:   []string{"0.0.1", "v0.0.1", "podinfo/v6.6.3"},
		},
		{
			URL:             "https://github.com/errordeveloper/tape-git-testing",
			CheckoutTag:     "v0.0.2",
			LoadPath:        "podinfo",
			ExpectManifests: []string{"kustomization.yaml", "deployment.yaml", "hpa.yaml", "service.yaml"},
			ExpectImageTags: []string{"v6.7.0"},
			ExpectRawTags:   []string{"0.0.2", "v0.0.2", "podinfo/v6.7.0"},
		},
		{
			URL:             "https://github.com/errordeveloper/tape-git-testing",
			CheckoutHash:    "9eeeed9f4ff44812ca23dba1bd0af9f509686d21", // => v0.0.1
			LoadPath:        "podinfo",
			ExpectManifests: []string{"kustomization.yaml", "deployment.yaml", "hpa.yaml", "service.yaml"},
			ExpectImageTags: []string{"v6.6.3"},
			ExpectRawTags:   []string{"0.0.1", "v0.0.1", "podinfo/v6.6.3"},
		},
	}

	repos := &repos{}
	repos.init()
	defer repos.cleanup()

	for i := range testCases {
		tc := testCases[i]
		t.Run(tc.Name(), makeVCSTest(repos, tc))
	}
}

func makeVCSTest(repos *repos, tc vcsTestCase) func(t *testing.T) {
	return func(t *testing.T) {
		g := NewWithT(t)

		ctx := context.Background()
		checkoutPath, err := repos.clone(ctx, tc)
		g.Expect(err).NotTo(HaveOccurred())

		loadPath := filepath.Join(checkoutPath, tc.LoadPath)
		loader := loader.NewRecursiveManifestDirectoryLoader(loadPath)
		g.Expect(loader.Load()).To(Succeed())

		repoDetected, attreg, err := DetectVCS(loadPath)
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(repoDetected).To(BeTrue())
		g.Expect(attreg).ToNot(BeNil())

		scanner := imagescanner.NewDefaultImageScanner()
		scanner.WithProvinanceAttestor(attreg)

		if tc.ExpectManifests != nil {
			g.Expect(loader.Paths()).To(HaveLen(len(tc.ExpectManifests)))
			for _, manifest := range tc.ExpectManifests {
				g.Expect(loader.ContainsRelPath(manifest)).To(BeTrue())
			}
		}

		g.Expect(scanner.Scan(loader.RelPaths())).To(Succeed())

		collection, err := attreg.MakePathCheckSummarySummaryCollection()
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(collection).ToNot(BeNil())
		g.Expect(collection.Providers).To(ConsistOf("git"))
		g.Expect(collection.EntryGroups).To(HaveLen(1))
		g.Expect(collection.EntryGroups[0]).To(HaveLen(5))

		vcsSummary := attreg.BaseDirSummary()
		g.Expect(vcsSummary).ToNot(BeNil())
		summaryJSON, err := json.Marshal(vcsSummary.Full())
		g.Expect(err).NotTo(HaveOccurred())
		t.Logf("VCS info for %q: %s", tc.LoadPath, summaryJSON)

		g.Expect(attreg.AssociateCoreStatements()).To(Succeed())

		statements := attreg.GetStatements()
		g.Expect(statements).To(HaveLen(1))
		g.Expect(statements[0].GetSubject()).To(HaveLen(4))

		// TODO: validate schema

		groupSummary, ok := vcsSummary.Full().(*git.Summary)
		g.Expect(ok).To(BeTrue())
		ref := groupSummary.Git.Reference
		g.Expect(ref.Tags).To(HaveLen(len(tc.ExpectRawTags)))
		imageTagNames := make([]string, len(ref.Tags))
		for i, tag := range ref.Tags {
			imageTagNames[i] = tag.Name
		}
		g.Expect(imageTagNames).To(ConsistOf(tc.ExpectRawTags))

		image, err := name.NewRepository("podinfo")
		g.Expect(err).NotTo(HaveOccurred())

		semVerTags := oci.SemVerTagsFromAttestations(ctx, image.Tag("test.123456"), statements...)
		g.Expect(semVerTags).To(HaveLen(len(tc.ExpectImageTags)))
		semVerTagNames := make([]string, len(semVerTags))
		for i, tag := range semVerTags {
			semVerTagNames[i] = tag.TagStr()
		}
		g.Expect(semVerTagNames).To(ConsistOf(tc.ExpectImageTags))
	}
}

type repos struct {
	workDir string
	tempDir string
	cache   map[string]string
}

func (r *repos) init() error {
	workDir, err := os.Getwd()
	if err != nil {
		return err
	}
	r.workDir = workDir
	tempDir, err := os.MkdirTemp("", ".vcs-test-*")
	if err != nil {
		return err
	}
	r.tempDir = tempDir
	r.cache = map[string]string{}
	return nil
}

func (r *repos) cleanup() error {
	if r.tempDir == "" {
		return nil
	}
	return os.RemoveAll(r.tempDir)
}

func (r *repos) mktemp() (string, error) {
	return os.MkdirTemp(r.tempDir, "repo-*")
}

func (r *repos) mirror(ctx context.Context, tc vcsTestCase) (string, error) {
	if _, ok := r.cache[tc.URL]; !ok {
		mirrorDir, err := r.mktemp()
		if err != nil {
			return "", err
		}
		_, err = gogit.PlainCloneContext(ctx, mirrorDir, true, &gogit.CloneOptions{Mirror: true, URL: tc.URL})
		if err != nil {
			return "", err
		}
		r.cache[tc.URL] = mirrorDir
	}
	return r.cache[tc.URL], nil
}

func (r *repos) clone(ctx context.Context, tc vcsTestCase) (string, error) {
	mirrorDir, err := r.mirror(ctx, tc)
	if err != nil {
		return "", err
	}
	checkoutDir, err := r.mktemp()
	if err != nil {
		return "", err
	}

	opts := &gogit.CloneOptions{URL: mirrorDir}
	if tc.CheckoutTag != "" {
		opts.ReferenceName = plumbing.NewTagReferenceName(tc.CheckoutTag)
	} else if tc.Branch != "" {
		opts.ReferenceName = plumbing.NewBranchReferenceName(tc.Branch)
	}

	repo, err := gogit.PlainCloneContext(ctx, checkoutDir, false, opts)
	if err != nil {
		return "", fmt.Errorf("failed to clone: %w", err)
	}

	if tc.CheckoutHash != "" {
		workTree, err := repo.Worktree()
		if err != nil {
			return "", err
		}
		opts := &gogit.CheckoutOptions{
			Hash: plumbing.NewHash(tc.CheckoutHash),
		}

		if err := workTree.Checkout(opts); err != nil {
			return "", err
		}
	}
	return filepath.Rel(r.workDir, checkoutDir)
}
