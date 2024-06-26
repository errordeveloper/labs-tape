# Tape is for packaging applications

## What is Tape?

Tape is a tool that can package an entire application as a self-contained (taped) OCI image that can be deployed to a
Kubernetes cluster. A taped OCI image contains all application components and Kubernetes resources required to run all
of the components together.

## What problem does it solve?

The process of building and deploying an application to Kubernetes is highly fragmented. There are many choices that
one has to make when implementing CI/CD. Leaving aside what CI vendor you pick, the starting point is the overall design of
a pipeline, repository structure, Dockerfiles, use of OCI registry, mapping of revisions to image tags, rules of deployment
based on tags, promotion between environments, as well as various kinds of ways to manage Kubernetes configs and how these
configs get deployed to any particular cluster.

Of course, there is some essential complexity, all CI pipelines will be different and everyone cannot use one and the same
tool for philosophical reasons. However, as most CI systems rely on shell scripts, it's quite challenging to define
contracts. What Tape does is all about having a simple artifact, which is a concrete contractual term, and Tape ensures
it has certain properties, for example, it references all formal dependencies by digest, i.e. dependencies that are OCI images
referenced from a canonical `image` field in a Kubernetes resource (e.g. `spec.containers[].image`).

Another issue that arises due to fragmentation is about collaboration between organisations that have made different choices,
despite all using OCI images and Kubernetes APIs.  To illustrate this, let's ask ourselves: "Why should there be any difference
in the mechanics of how my application is deployed to a Kubernetes cluster vs. someone else's application being deployed to the
same cluster?". Of course, it all comes down to Kubernetes API resources, but there is often a lot that happens in between.
Sometimes Helm is used, sometimes it's Kustomize, sometimes plain manifests are used with some bespoke automation around configs Git,
and there many examples of configs defined in scripting languages as well. Of course, it's very often a mix of a few approaches.
Even without anything too complicated, there is no such thing as a typical setup with Helm or Kustomize. The problem is that
the knowledge of what the choices are exactly and how some well-known tools might be used is not trivial or automatically
transferable, and there is little to say about the benefits of having something special.

Tape addresses the complexity by using OCI for the distribution of runnable code as well configuration required to run it
correctly on Kubernetes. Tape introduces a notion of application artifact without introducing any particular model of how
components are composed into an artifact. This model aims to be a layer of interoperability between different tools and
provide a logical supply chain entry point and location for storing metadata.

The best analogy is flatpack furniture. Presently, deployment of an application is as if flatpack hasn't been invented, so
when someone orders a wooden cabinet, all that arrives in a box is just the pieces of wood, they have to shop for nuts,
bolts, and tools. Of course, that might be desirable for some, as they have a well-stocked workshop with the best tools and
a decent selection of nuts and bolts. But did the box even include assembly instructions with the list of nuts and bolts
one has to buy?
That model doesn't scale to the consumer market. Of course, some consumers might have a toolbox, but very few will be able
to find the right nuts and bolts or even bother looking for any, they might just send the whole box back instead.
A taped image is like a flatpack package, it has everything needed as well as assembly instructions, without introducing
new complexity elsewhere and allows users to keep using their favourite tools.

To summarise, Tape reduces the complexity of application delivery by packaging the entire application as an OCI image, providing
a transferable artifact that includes config and all of the components, analogous to flatpack furniture. This notion of artifact
is very important because it helps to define a concise contract.
Tape also produces attestations about the provenance of the configuration source as well as any transforms it applies to the
source. The attestations are attached to the resulting OCI image, so it helps with security and observability as well.

## How does Tape work?

Tape can parse a directory with Kubernetes configuration and find all canonical references to application images.
If an image reference contains a digest, Tape will use it, otherwise it resolves it by making a registry API call.
For each of the images, Tape searches of all well-known related tags, such as external signatures, attestations and
SBOMs. Tape will make a copy of every application image and any tags related to it to a registry the user has specified.
Once images are copied, it updates manifests with new references and bundles the result in an OCI artifact pushed to
the same repo in the registry.

Copying of all application images and referencing by digest is performed to ensure the application and its configuration
are tightly coupled together to provide a single link in the supply chain as well as a single point of distribution
and access control for the whole application.

Tape also checks the VCS provenance of manifests, so if any manifest files are checked in Git, Tape will attest to what
Git repository each file came from, all of the revision metadata, and whether it's been modified or not.
Additionally, Tape attests to all key steps that it performs, e.g. original image references it detects and manifest
checksums. It stores the attestations using in-toto format in an OCI artifact.

## Usage

Tape has the following commands:

- `tape images` - examine images referenced by a given set of manifests before packaging them
- `tape package` - package an artifact and push it to a registry
- `tape pull` – download and extract contents and attestations from an existing artifact
- `tape view` – inspect an existing artifact

### Example

First, clone the repo and build `tape` binary:

```console
git clone -q git@github.com:errordeveloper/tape.git ; cd ./labs-brown-tape
(cd ./tape ; go build)
```

Clone podinfo app repo:
```console
(git clone -q https://github.com/stefanprodan/podinfo ; cd podinfo ; git switch --detach 6.7.0)
```

Examine podinfo manifests:
```console
$ ./tape/tape images --output-format text --manifest-dir ./podinfo/kustomize
INFO[0000] resolving image digests
INFO[0000] resolving related images
ghcr.io/stefanprodan/podinfo:6.7.0@sha256:d2b3cd93a48acdc91327533ce28fcb3169b2d9feaf73817dc2eb68858df64edb
  Alias: podinfo
  Sources:
    ghcr.io/stefanprodan/podinfo:6.7.0 deployment.yaml:26:16@sha256:5dcd7a6bd78c6d3613eefdea4747d2ba7e251ee355d793e165c9862ca7d69c9c
  Digest provided: false
  OCI manifests:
    sha256:0060e4fc3052c383ea920673d08388fd6aa3bfc3536932f7c08edd4e4a616520  application/vnd.oci.image.manifest.v1+json  linux/amd64  1625
    sha256:3ce55f0cfdb1200738bd3d0cd955866f09fe84edbb5930575a155beda649bde7  application/vnd.oci.image.manifest.v1+json  linux/arm/v7  1625
    sha256:53d266d18dcd714920b0aed2c54b55d85bbf38ff59b931b4d6f6b6aca0da7e7d  application/vnd.oci.image.manifest.v1+json  linux/arm64  1625
    sha256:064e58b3828b21d6c967c44deaab8a45748de84d137a4452a612a17979ade8c8  application/vnd.oci.image.manifest.v1+json  unknown/unknown  840
    sha256:2bdce4c7c136cb08f3f4a8425dd0ab3fe5f8f76c8a2d4d98f23d6f60e52358c0  application/vnd.oci.image.manifest.v1+json  unknown/unknown  840
    sha256:bf839dfebc879e6d39825ee2386bc3590d6d248a22282e0b268c38bde2cae874  application/vnd.oci.image.manifest.v1+json  unknown/unknown  840
  External attestations: 0
  Inline SBOMs: 0
  External SBOMs: 0
  Inline signatures: 0
  External signatures: 1
  Inline attestations: 0
$
```

Package podinfo:
```console
$ INFO[0008] VCS info for "./podinfo/kustomize": {"unmodified":true,"path":"kustomize","uri":"https://github.com/stefanprodan/podinfo","isDir":true,"git":{"object":{"treeHash":"3f1d5f59fce5f67017dd007fe00131f6123896cf"},"remotes":{"origin":["https://github.com/stefanprodan/podinfo"]},"reference":{"name":"HEAD","hash":"0b1481aa8ed0a6c34af84f779824a74200d5c1d6","type":"hash-reference","tags":[{"name":"6.7.0","hash":"b0f4c201b5bf1923a0c04c7d2da3b8d66cd18e39","target":"0b1481aa8ed0a6c34af84f779824a74200d5c1d6","signature":{"pgp":"LS0tLS1CRUdJTiBQR1AgU0lHTkFUVVJFLS0tLS0KCmlIVUVBQkVJQUIwV0lRUUhnRXhVcjRGckxkS3pwTll5bWE2dzVBaGJyd1VDWm5ocFN3QUtDUkF5bWE2dzVBaGIKcitoZ0FQNDVSeStsOXlQQ1RKd1VER1lqMG9hZjZIZTZlcUZjNnQ2RjdpdkViazkwZVFEK09WYVN4Rm5LdGsyawpxR1Q3S0dZdy80ZlBJeDVnYUlIUFVzbksvR3B6azZNPQo9NllwZwotLS0tLUVORCBQR1AgU0lHTkFUVVJFLS0tLS0K","validated":false}}],"signature":{"pgp":"LS0tLS1CRUdJTiBQR1AgU0lHTkFUVVJFLS0tLS0KCndzRmNCQUFCQ0FBUUJRSm1lR2s5Q1JDMWFRN3V1NVVobEFBQUN2Z1FBR1drUG4xbXI2bGViaVlsRjJwZTRYY3gKVENQMkRQQVBnM2NnWExFQkt6dndlU0NCRXE1Zi9UaEVOT3lhOWlxYUFvQmRkM2Z6c1YrekdlTmpYRXFRTHI0dApuZkY1cE40THg4R2V2bzNyeXBESnlBd29kR2xFdEFEcjFrbEthSDFKSWZNdkFxaEExcmZMVElmQ01nTTNLVEF6CmczaHhOVXN0TnJoY2JETUZtYUVzaU5NeFRlTXBRSmlJcjBSRmJnVXcxWkhoU0NydzhISGxydjE5VC8wUWFXcnQKR1pjaHdzd2NOUHVEZm4vRjRCZlNEV2F2MjBUandMTmlxY2lndFgrQzBScVRqZkhVWnhjcSt4eWpCaTNnWHpTNQp0bGxEVzdneVJvR2pDdTAyZ2FiV3YwcnFtVUx1cHpHYTBYRUl6M0J5bDVwOXl3WFY3d1czdERjVkNLRU9YMzBnCnR2aWNkQ3B0Zjl2NlovZHRSNG1YM2lVeUdYRlZkM01xY2ptaGlqcEFJbW0yVXJORElVUVFRVkpkMzlGeE9uOVEKU29qd3N1b1FXYUhXbEdkWTdEWTVia2ZYM1cvNXNSZjlBVjdUemV5VjQvcmVnZ3EzSGY5ZC9QZEtaVEdLRXh3bwpUbXVNcTVUdmdVMWNjUTUzSkxrNVFGS0lzVmErZEh4SXd5YW5rcWNIbFBCRXlSSmtJajZlSVZRVldNQndCYUNhCkIvSnoxeGcweWc2SS9EZnRUQVdwTlBMUFQ2eFNBM09IZnZqQm1NblZGMzN0YnQ3Y3JtYVg0Wm5IbndZazZESVQKSUFNRkM4ay82OUZ6SFVRSWJJVTQrQk5CbVRpaTdnS2Y0cDlWNGRBNk95cTZFU1lkVlg2L2Z0OUNLMUY4eGxrQQpvSDNHMTI1dFdQQ0ZrTk9jb3ZGLwo9UnFzNwotLS0tLUVORCBQR1AgU0lHTkFUVVJFLS0tLS0KCg==","validated":false}}}} 
INFO[0026] resolving image digests
INFO[0027] resolving related images
INFO[0033] copying images
INFO[0087] copied images: ttl.sh/tape/podinfo:app.e1053e7cf3b59bec02980dd5cbbbb0cd151056e32810d25ee67ef52cb388a53f@sha256:d2b3cd93a48acdc91327533ce28fcb3169b2d9feaf73817dc2eb68858df64edb, ttl.sh/tape/podinfo:sha256-d2b3cd93a48acdc91327533ce28fcb3169b2d9feaf73817dc2eb68858df64edb.sig@sha256:ec3fea5d536913f3e1c30e9475279a32abeedb5bb2cb287ab90490f5388bdaf3, ttl.sh/tape/podinfo:app.e3f1195ebee343fd04dc8c447e98d1c85e8e9626fbb5078c0548d28dbb8153da@sha256:6601559170a45bf4fcecd113d1eba107d2d552adf58b389ccde39bf669ffa65a
INFO[0012] updating manifest files
INFO[0095] created package "ttl.sh/tape/podinfo:6.7.0@sha256:d42b82bcdeba762ce56c01ac36270df96bc4d4d37881fb866504305f942e55d1"
$
```

Store image name and config tag+digest as environment variables:
```console
podinfo_image="ttl.sh/tape/podinfo"
podinfo_config="${podinfo_image}:6.7.0@sha256:d42b82bcdeba762ce56c01ac36270df96bc4d4d37881fb866504305f942e55d1"
```

Examine the OCI index of the config image that's been created:
```console
$ crane manifest "${podinfo_config}" | jq .
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.index.v1+json",
  "manifests": [
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "size": 625,
      "digest": "sha256:cac65fe10f725537d5d7393a0cd9ba15e7bb72d1029ab3a522b532e29f13743c",
      "platform": {
        "architecture": "unknown",
        "os": "unknown"
      },
      "artifactType": "application/vnd.docker.tape.content.v1alpha1.tar+gzip"
    },
    {
      "mediaType": "application/vnd.oci.image.manifest.v1+json",
      "size": 1661,
      "digest": "sha256:d9ab1a127981988fc82b70fad8079af9444f24b65580baca349df8032c51519a",
      "platform": {
        "architecture": "unknown",
        "os": "unknown"
      },
      "artifactType": "application/vnd.docker.tape.attest.v1alpha1.jsonl+gzip"
    }
  ],
  "annotations": {
    "org.opencontainers.image.created": "2023-08-30T11:05:44+01:00"
  }
}
$
```
Examine each of the two 2nd-level OCI manifests, the first one is for config contents, and the second for attestations:
```console
$ crane manifest "${podinfo_image}@$(crane manifest "${podinfo_config}" | jq -r '.manifests[0].digest')" | jq .
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": {
    "mediaType": "application/vnd.docker.tape.content.v1alpha1.tar+gzip",
    "size": 233,
    "digest": "sha256:fddda5c5f84b9041f1746f13ecb63bab559129228363fdb9e8b826aa6953be4e"
  },
  "layers": [
    {
      "mediaType": "application/vnd.docker.tape.content.v1alpha1.tar+gzip",
      "size": 1211,
      "digest": "sha256:75967cdbb7c0a3439b46247b09c3d528dabb261586c12fa2f7b5df888baa238f"
    }
  ],
  "annotations": {
    "application/vnd.docker.tape.content-interpreter.v1alpha1": "application/vnd.docker.tape.kubectl-apply.v1alpha1.tar+gzip",
    "org.opencontainers.image.created": "2023-08-30T11:05:44+01:00"
  }
}
$ crane manifest "${podinfo_image}@$(crane manifest "${podinfo_config}" | jq -r '.manifests[1].digest')" | jq .
{
  "schemaVersion": 2,
  "mediaType": "application/vnd.oci.image.manifest.v1+json",
  "config": {
    "mediaType": "application/vnd.docker.tape.attest.v1alpha1.jsonl+gzip",
    "size": 233,
    "digest": "sha256:87b06d185f03315a2369ebbcc1d4e9a325e245735fd4e997af550bc4290f8338"
  },
  "layers": [
    {
      "mediaType": "application/vnd.docker.tape.attest.v1alpha1.jsonl+gzip",
      "size": 2332,
      "digest": "sha256:5bebe5e0d68cfb1202178a096707e4b79ca3d9569e6d40a1084b7c094430bbb0"
    }
  ],
  "annotations": {
    "application/vnd.docker.tape.attestations-summary.v1alpha1": "eyJudW1TdGFtZW50ZXMiOjQsInByZWRpY2F0ZVR5cGVzIjpbImRvY2tlci5jb20vdGFwZS9NYW5pZmVzdERpci92MC4yIiwiZG9ja2VyLmNvbS90YXBlL09yaWdpbmFsSW1hZ2VSZWYvdjAuMSIsImRvY2tlci5jb20vdGFwZS9SZXBsYWNlZEltYWdlUmVmL3YwLjEiLCJkb2NrZXIuY29tL3RhcGUvUmVzb2x2ZWRJbWFnZVJlZi92MC4xIl0sInN1YmplY3QiOlt7Im5hbWUiOiJrdXN0b21pemUvZGVwbG95bWVudC55YW1sIiwiZGlnZXN0Ijp7InNoYTI1NiI6IjVkY2Q3YTZiZDc4YzZkMzYxM2VlZmRlYTQ3NDdkMmJhN2UyNTFlZTM1NWQ3OTNlMTY1Yzk4NjJjYTdkNjljOWMifX0seyJuYW1lIjoia3VzdG9taXplL2RlcGxveW1lbnQueWFtbCIsImRpZ2VzdCI6eyJzaGEyNTYiOiJhZDg0ZGJkMTQ1MGE3NTJhZTZhZDFkOTkyNTAzMTBiZWNhOGZlNzAzMWRhNjlmNzJhN2EzOGZkOTVmMDJmZTM4In19LHsibmFtZSI6Imt1c3RvbWl6ZS9ocGEueWFtbCIsImRpZ2VzdCI6eyJzaGEyNTYiOiJkNGIyZmY2YWY2MDc3ZDA2MDY1MmI5ODQ5ZDBjZGFiMWUyODE4YzY0ZTViNTAxZDQxNGQ4OGMyNGQ1YWJkZWY4In19LHsibmFtZSI6Imt1c3RvbWl6ZS9rdXN0b21pemF0aW9uLnlhbWwiLCJkaWdlc3QiOnsic2hhMjU2IjoiODBkNjUyZTdmYTc2ZDQ3YTY1NWE2YmNkZDA5NmIyZmMzMmEzNTJiZjUzZmRjMTVlNjNlYTQ3ZTE5YTBiMTU1YyJ9fSx7Im5hbWUiOiJrdXN0b21pemUvc2VydmljZS55YW1sIiwiZGlnZXN0Ijp7InNoYTI1NiI6ImYxODc1NjZmMjEyZmMxNGU5YmU2M2RhYjc5ZDlkZjVjZmE3MWRjMjg0NTA5ZjIyN2U5YTQyNWQxNTJmZWVjODUifX1dfQo=",
    "org.opencontainers.image.created": "2023-08-30T11:05:44+01:00"
  }
}
$
```

Store digests as variables:
```console
tape_config_digest="$(crane manifest "${podinfo_image}@$(crane manifest "${podinfo_config}" | jq -r '.manifests[0].digest')" | jq -r '.layers[0].digest')"
tape_attest_digest="$(crane manifest "${podinfo_image}@$(crane manifest "${podinfo_config}" | jq -r '.manifests[1].digest')" | jq -r '.layers[0].digest')"
```

Examine config contents:
```console
$ crane blob ${podinfo_image}@${tape_config_digest} | tar t
.
deployment.yaml
hpa.yaml
kustomization.yaml
service.yaml
$
```

Examine attestations:
```console
$ crane blob ${podinfo_image}@${tape_attest_digest} | gunzip | jq .
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "predicateType": "docker.com/tape/ManifestDir/v0.2",
  "subject": [
    {
      "name": "kustomize/deployment.yaml",
      "digest": {
        "sha256": "5dcd7a6bd78c6d3613eefdea4747d2ba7e251ee355d793e165c9862ca7d69c9c"
      }
    },
    {
      "name": "kustomize/hpa.yaml",
      "digest": {
        "sha256": "d4b2ff6af6077d060652b9849d0cdab1e2818c64e5b501d414d88c24d5abdef8"
      }
    },
    {
      "name": "kustomize/kustomization.yaml",
      "digest": {
        "sha256": "80d652e7fa76d47a655a6bcdd096b2fc32a352bf53fdc15e63ea47e19a0b155c"
      }
    },
    {
      "name": "kustomize/service.yaml",
      "digest": {
        "sha256": "f187566f212fc14e9be63dab79d9df5cfa71dc284509f227e9a425d152feec85"
      }
    }
  ],
  "predicate": {
    "containedInDirectory": {
      "path": "kustomize",
      "vcsEntries": {
        "providers": [
          "git"
        ],
        "entryGroups": [
          [
            {
              "unmodified": true,
              "path": "kustomize",
              "uri": "https://github.com/stefanprodan/podinfo",
              "isDir": true,
              "git": {
                "object": {
                  "treeHash": "3f1d5f59fce5f67017dd007fe00131f6123896cf"
                },
                "remotes": {
                  "origin": [
                    "https://github.com/stefanprodan/podinfo"
                  ]
                },
                "reference": {
                  "name": "HEAD",
                  "hash": "0b1481aa8ed0a6c34af84f779824a74200d5c1d6",
                  "type": "hash-reference",
                  "tags": [
                    {
                      "name": "6.7.0",
                      "hash": "b0f4c201b5bf1923a0c04c7d2da3b8d66cd18e39",
                      "target": "0b1481aa8ed0a6c34af84f779824a74200d5c1d6",
                      "signature": {
                        "pgp": "LS0tLS1CRUdJTiBQR1AgU0lHTkFUVVJFLS0tLS0KCmlIVUVBQkVJQUIwV0lRUUhnRXhVcjRGckxkS3pwTll5bWE2dzVBaGJyd1VDWm5ocFN3QUtDUkF5bWE2dzVBaGIKcitoZ0FQNDVSeStsOXlQQ1RKd1VER1lqMG9hZjZIZTZlcUZjNnQ2RjdpdkViazkwZVFEK09WYVN4Rm5LdGsyawpxR1Q3S0dZdy80ZlBJeDVnYUlIUFVzbksvR3B6azZNPQo9NllwZwotLS0tLUVORCBQR1AgU0lHTkFUVVJFLS0tLS0K",
                        "validated": false
                      }
                    }
                  ],
                  "signature": {
                    "pgp": "LS0tLS1CRUdJTiBQR1AgU0lHTkFUVVJFLS0tLS0KCndzRmNCQUFCQ0FBUUJRSm1lR2s5Q1JDMWFRN3V1NVVobEFBQUN2Z1FBR1drUG4xbXI2bGViaVlsRjJwZTRYY3gKVENQMkRQQVBnM2NnWExFQkt6dndlU0NCRXE1Zi9UaEVOT3lhOWlxYUFvQmRkM2Z6c1YrekdlTmpYRXFRTHI0dApuZkY1cE40THg4R2V2bzNyeXBESnlBd29kR2xFdEFEcjFrbEthSDFKSWZNdkFxaEExcmZMVElmQ01nTTNLVEF6CmczaHhOVXN0TnJoY2JETUZtYUVzaU5NeFRlTXBRSmlJcjBSRmJnVXcxWkhoU0NydzhISGxydjE5VC8wUWFXcnQKR1pjaHdzd2NOUHVEZm4vRjRCZlNEV2F2MjBUandMTmlxY2lndFgrQzBScVRqZkhVWnhjcSt4eWpCaTNnWHpTNQp0bGxEVzdneVJvR2pDdTAyZ2FiV3YwcnFtVUx1cHpHYTBYRUl6M0J5bDVwOXl3WFY3d1czdERjVkNLRU9YMzBnCnR2aWNkQ3B0Zjl2NlovZHRSNG1YM2lVeUdYRlZkM01xY2ptaGlqcEFJbW0yVXJORElVUVFRVkpkMzlGeE9uOVEKU29qd3N1b1FXYUhXbEdkWTdEWTVia2ZYM1cvNXNSZjlBVjdUemV5VjQvcmVnZ3EzSGY5ZC9QZEtaVEdLRXh3bwpUbXVNcTVUdmdVMWNjUTUzSkxrNVFGS0lzVmErZEh4SXd5YW5rcWNIbFBCRXlSSmtJajZlSVZRVldNQndCYUNhCkIvSnoxeGcweWc2SS9EZnRUQVdwTlBMUFQ2eFNBM09IZnZqQm1NblZGMzN0YnQ3Y3JtYVg0Wm5IbndZazZESVQKSUFNRkM4ay82OUZ6SFVRSWJJVTQrQk5CbVRpaTdnS2Y0cDlWNGRBNk95cTZFU1lkVlg2L2Z0OUNLMUY4eGxrQQpvSDNHMTI1dFdQQ0ZrTk9jb3ZGLwo9UnFzNwotLS0tLUVORCBQR1AgU0lHTkFUVVJFLS0tLS0KCg==",
                    "validated": false
                  }
                }
              }
            },
            {
              "unmodified": true,
              "path": "kustomize/deployment.yaml",
              "uri": "https://github.com/stefanprodan/podinfo",
              "digest": {
                "sha256": "5dcd7a6bd78c6d3613eefdea4747d2ba7e251ee355d793e165c9862ca7d69c9c"
              },
              "git": {
                "object": {
                  "treeHash": "0045078cfd7ba0b31492b814c18f191aeffef3cd",
                  "commitHash": "ff32a1fc4b45b2fd2850e7204e5af7ef44ea1c73"
                },
                "remotes": {
                  "origin": [
                    "https://github.com/stefanprodan/podinfo"
                  ]
                },
                "reference": {
                  "name": "HEAD",
                  "hash": "0b1481aa8ed0a6c34af84f779824a74200d5c1d6",
                  "type": "hash-reference",
                  "tags": [
                    {
                      "name": "6.7.0",
                      "hash": "b0f4c201b5bf1923a0c04c7d2da3b8d66cd18e39",
                      "target": "0b1481aa8ed0a6c34af84f779824a74200d5c1d6",
                      "signature": {
                        "pgp": "LS0tLS1CRUdJTiBQR1AgU0lHTkFUVVJFLS0tLS0KCmlIVUVBQkVJQUIwV0lRUUhnRXhVcjRGckxkS3pwTll5bWE2dzVBaGJyd1VDWm5ocFN3QUtDUkF5bWE2dzVBaGIKcitoZ0FQNDVSeStsOXlQQ1RKd1VER1lqMG9hZjZIZTZlcUZjNnQ2RjdpdkViazkwZVFEK09WYVN4Rm5LdGsyawpxR1Q3S0dZdy80ZlBJeDVnYUlIUFVzbksvR3B6azZNPQo9NllwZwotLS0tLUVORCBQR1AgU0lHTkFUVVJFLS0tLS0K",
                        "validated": false
                      }
                    }
                  ],
                  "signature": {
                    "pgp": "LS0tLS1CRUdJTiBQR1AgU0lHTkFUVVJFLS0tLS0KCndzRmNCQUFCQ0FBUUJRSm1lR2s5Q1JDMWFRN3V1NVVobEFBQUN2Z1FBR1drUG4xbXI2bGViaVlsRjJwZTRYY3gKVENQMkRQQVBnM2NnWExFQkt6dndlU0NCRXE1Zi9UaEVOT3lhOWlxYUFvQmRkM2Z6c1YrekdlTmpYRXFRTHI0dApuZkY1cE40THg4R2V2bzNyeXBESnlBd29kR2xFdEFEcjFrbEthSDFKSWZNdkFxaEExcmZMVElmQ01nTTNLVEF6CmczaHhOVXN0TnJoY2JETUZtYUVzaU5NeFRlTXBRSmlJcjBSRmJnVXcxWkhoU0NydzhISGxydjE5VC8wUWFXcnQKR1pjaHdzd2NOUHVEZm4vRjRCZlNEV2F2MjBUandMTmlxY2lndFgrQzBScVRqZkhVWnhjcSt4eWpCaTNnWHpTNQp0bGxEVzdneVJvR2pDdTAyZ2FiV3YwcnFtVUx1cHpHYTBYRUl6M0J5bDVwOXl3WFY3d1czdERjVkNLRU9YMzBnCnR2aWNkQ3B0Zjl2NlovZHRSNG1YM2lVeUdYRlZkM01xY2ptaGlqcEFJbW0yVXJORElVUVFRVkpkMzlGeE9uOVEKU29qd3N1b1FXYUhXbEdkWTdEWTVia2ZYM1cvNXNSZjlBVjdUemV5VjQvcmVnZ3EzSGY5ZC9QZEtaVEdLRXh3bwpUbXVNcTVUdmdVMWNjUTUzSkxrNVFGS0lzVmErZEh4SXd5YW5rcWNIbFBCRXlSSmtJajZlSVZRVldNQndCYUNhCkIvSnoxeGcweWc2SS9EZnRUQVdwTlBMUFQ2eFNBM09IZnZqQm1NblZGMzN0YnQ3Y3JtYVg0Wm5IbndZazZESVQKSUFNRkM4ay82OUZ6SFVRSWJJVTQrQk5CbVRpaTdnS2Y0cDlWNGRBNk95cTZFU1lkVlg2L2Z0OUNLMUY4eGxrQQpvSDNHMTI1dFdQQ0ZrTk9jb3ZGLwo9UnFzNwotLS0tLUVORCBQR1AgU0lHTkFUVVJFLS0tLS0KCg==",
                    "validated": false
                  }
                }
              }
            },
            {
              "unmodified": true,
              "path": "kustomize/hpa.yaml",
              "uri": "https://github.com/stefanprodan/podinfo",
              "digest": {
                "sha256": "d4b2ff6af6077d060652b9849d0cdab1e2818c64e5b501d414d88c24d5abdef8"
              },
              "git": {
                "object": {
                  "treeHash": "263e9128848695fec5ab76c7f864b11ec98c2149",
                  "commitHash": "607303dca9fee3d97e1bd78f9997512bcb78da42"
                },
                "remotes": {
                  "origin": [
                    "https://github.com/stefanprodan/podinfo"
                  ]
                },
                "reference": {
                  "name": "HEAD",
                  "hash": "0b1481aa8ed0a6c34af84f779824a74200d5c1d6",
                  "type": "hash-reference",
                  "tags": [
                    {
                      "name": "6.7.0",
                      "hash": "b0f4c201b5bf1923a0c04c7d2da3b8d66cd18e39",
                      "target": "0b1481aa8ed0a6c34af84f779824a74200d5c1d6",
                      "signature": {
                        "pgp": "LS0tLS1CRUdJTiBQR1AgU0lHTkFUVVJFLS0tLS0KCmlIVUVBQkVJQUIwV0lRUUhnRXhVcjRGckxkS3pwTll5bWE2dzVBaGJyd1VDWm5ocFN3QUtDUkF5bWE2dzVBaGIKcitoZ0FQNDVSeStsOXlQQ1RKd1VER1lqMG9hZjZIZTZlcUZjNnQ2RjdpdkViazkwZVFEK09WYVN4Rm5LdGsyawpxR1Q3S0dZdy80ZlBJeDVnYUlIUFVzbksvR3B6azZNPQo9NllwZwotLS0tLUVORCBQR1AgU0lHTkFUVVJFLS0tLS0K",
                        "validated": false
                      }
                    }
                  ],
                  "signature": {
                    "pgp": "LS0tLS1CRUdJTiBQR1AgU0lHTkFUVVJFLS0tLS0KCndzRmNCQUFCQ0FBUUJRSm1lR2s5Q1JDMWFRN3V1NVVobEFBQUN2Z1FBR1drUG4xbXI2bGViaVlsRjJwZTRYY3gKVENQMkRQQVBnM2NnWExFQkt6dndlU0NCRXE1Zi9UaEVOT3lhOWlxYUFvQmRkM2Z6c1YrekdlTmpYRXFRTHI0dApuZkY1cE40THg4R2V2bzNyeXBESnlBd29kR2xFdEFEcjFrbEthSDFKSWZNdkFxaEExcmZMVElmQ01nTTNLVEF6CmczaHhOVXN0TnJoY2JETUZtYUVzaU5NeFRlTXBRSmlJcjBSRmJnVXcxWkhoU0NydzhISGxydjE5VC8wUWFXcnQKR1pjaHdzd2NOUHVEZm4vRjRCZlNEV2F2MjBUandMTmlxY2lndFgrQzBScVRqZkhVWnhjcSt4eWpCaTNnWHpTNQp0bGxEVzdneVJvR2pDdTAyZ2FiV3YwcnFtVUx1cHpHYTBYRUl6M0J5bDVwOXl3WFY3d1czdERjVkNLRU9YMzBnCnR2aWNkQ3B0Zjl2NlovZHRSNG1YM2lVeUdYRlZkM01xY2ptaGlqcEFJbW0yVXJORElVUVFRVkpkMzlGeE9uOVEKU29qd3N1b1FXYUhXbEdkWTdEWTVia2ZYM1cvNXNSZjlBVjdUemV5VjQvcmVnZ3EzSGY5ZC9QZEtaVEdLRXh3bwpUbXVNcTVUdmdVMWNjUTUzSkxrNVFGS0lzVmErZEh4SXd5YW5rcWNIbFBCRXlSSmtJajZlSVZRVldNQndCYUNhCkIvSnoxeGcweWc2SS9EZnRUQVdwTlBMUFQ2eFNBM09IZnZqQm1NblZGMzN0YnQ3Y3JtYVg0Wm5IbndZazZESVQKSUFNRkM4ay82OUZ6SFVRSWJJVTQrQk5CbVRpaTdnS2Y0cDlWNGRBNk95cTZFU1lkVlg2L2Z0OUNLMUY4eGxrQQpvSDNHMTI1dFdQQ0ZrTk9jb3ZGLwo9UnFzNwotLS0tLUVORCBQR1AgU0lHTkFUVVJFLS0tLS0KCg==",
                    "validated": false
                  }
                }
              }
            },
            {
              "unmodified": true,
              "path": "kustomize/kustomization.yaml",
              "uri": "https://github.com/stefanprodan/podinfo",
              "digest": {
                "sha256": "80d652e7fa76d47a655a6bcdd096b2fc32a352bf53fdc15e63ea47e19a0b155c"
              },
              "git": {
                "object": {
                  "treeHash": "f6a64bbfd5edefa203d3e639afc9cd2546c2778b",
                  "commitHash": "4c0dfaef0ee72480f7873f819657c83d78127bf5"
                },
                "remotes": {
                  "origin": [
                    "https://github.com/stefanprodan/podinfo"
                  ]
                },
                "reference": {
                  "name": "HEAD",
                  "hash": "0b1481aa8ed0a6c34af84f779824a74200d5c1d6",
                  "type": "hash-reference",
                  "tags": [
                    {
                      "name": "6.7.0",
                      "hash": "b0f4c201b5bf1923a0c04c7d2da3b8d66cd18e39",
                      "target": "0b1481aa8ed0a6c34af84f779824a74200d5c1d6",
                      "signature": {
                        "pgp": "LS0tLS1CRUdJTiBQR1AgU0lHTkFUVVJFLS0tLS0KCmlIVUVBQkVJQUIwV0lRUUhnRXhVcjRGckxkS3pwTll5bWE2dzVBaGJyd1VDWm5ocFN3QUtDUkF5bWE2dzVBaGIKcitoZ0FQNDVSeStsOXlQQ1RKd1VER1lqMG9hZjZIZTZlcUZjNnQ2RjdpdkViazkwZVFEK09WYVN4Rm5LdGsyawpxR1Q3S0dZdy80ZlBJeDVnYUlIUFVzbksvR3B6azZNPQo9NllwZwotLS0tLUVORCBQR1AgU0lHTkFUVVJFLS0tLS0K",
                        "validated": false
                      }
                    }
                  ],
                  "signature": {
                    "pgp": "LS0tLS1CRUdJTiBQR1AgU0lHTkFUVVJFLS0tLS0KCndzRmNCQUFCQ0FBUUJRSm1lR2s5Q1JDMWFRN3V1NVVobEFBQUN2Z1FBR1drUG4xbXI2bGViaVlsRjJwZTRYY3gKVENQMkRQQVBnM2NnWExFQkt6dndlU0NCRXE1Zi9UaEVOT3lhOWlxYUFvQmRkM2Z6c1YrekdlTmpYRXFRTHI0dApuZkY1cE40THg4R2V2bzNyeXBESnlBd29kR2xFdEFEcjFrbEthSDFKSWZNdkFxaEExcmZMVElmQ01nTTNLVEF6CmczaHhOVXN0TnJoY2JETUZtYUVzaU5NeFRlTXBRSmlJcjBSRmJnVXcxWkhoU0NydzhISGxydjE5VC8wUWFXcnQKR1pjaHdzd2NOUHVEZm4vRjRCZlNEV2F2MjBUandMTmlxY2lndFgrQzBScVRqZkhVWnhjcSt4eWpCaTNnWHpTNQp0bGxEVzdneVJvR2pDdTAyZ2FiV3YwcnFtVUx1cHpHYTBYRUl6M0J5bDVwOXl3WFY3d1czdERjVkNLRU9YMzBnCnR2aWNkQ3B0Zjl2NlovZHRSNG1YM2lVeUdYRlZkM01xY2ptaGlqcEFJbW0yVXJORElVUVFRVkpkMzlGeE9uOVEKU29qd3N1b1FXYUhXbEdkWTdEWTVia2ZYM1cvNXNSZjlBVjdUemV5VjQvcmVnZ3EzSGY5ZC9QZEtaVEdLRXh3bwpUbXVNcTVUdmdVMWNjUTUzSkxrNVFGS0lzVmErZEh4SXd5YW5rcWNIbFBCRXlSSmtJajZlSVZRVldNQndCYUNhCkIvSnoxeGcweWc2SS9EZnRUQVdwTlBMUFQ2eFNBM09IZnZqQm1NblZGMzN0YnQ3Y3JtYVg0Wm5IbndZazZESVQKSUFNRkM4ay82OUZ6SFVRSWJJVTQrQk5CbVRpaTdnS2Y0cDlWNGRBNk95cTZFU1lkVlg2L2Z0OUNLMUY4eGxrQQpvSDNHMTI1dFdQQ0ZrTk9jb3ZGLwo9UnFzNwotLS0tLUVORCBQR1AgU0lHTkFUVVJFLS0tLS0KCg==",
                    "validated": false
                  }
                }
              }
            },
            {
              "unmodified": true,
              "path": "kustomize/service.yaml",
              "uri": "https://github.com/stefanprodan/podinfo",
              "digest": {
                "sha256": "f187566f212fc14e9be63dab79d9df5cfa71dc284509f227e9a425d152feec85"
              },
              "git": {
                "object": {
                  "treeHash": "9450823d5a09afc116a37ee16da12f53a6f4836d",
                  "commitHash": "93e338a9641f1979039a517b23cca1a2f08d3dce"
                },
                "remotes": {
                  "origin": [
                    "https://github.com/stefanprodan/podinfo"
                  ]
                },
                "reference": {
                  "name": "HEAD",
                  "hash": "0b1481aa8ed0a6c34af84f779824a74200d5c1d6",
                  "type": "hash-reference",
                  "tags": [
                    {
                      "name": "6.7.0",
                      "hash": "b0f4c201b5bf1923a0c04c7d2da3b8d66cd18e39",
                      "target": "0b1481aa8ed0a6c34af84f779824a74200d5c1d6",
                      "signature": {
                        "pgp": "LS0tLS1CRUdJTiBQR1AgU0lHTkFUVVJFLS0tLS0KCmlIVUVBQkVJQUIwV0lRUUhnRXhVcjRGckxkS3pwTll5bWE2dzVBaGJyd1VDWm5ocFN3QUtDUkF5bWE2dzVBaGIKcitoZ0FQNDVSeStsOXlQQ1RKd1VER1lqMG9hZjZIZTZlcUZjNnQ2RjdpdkViazkwZVFEK09WYVN4Rm5LdGsyawpxR1Q3S0dZdy80ZlBJeDVnYUlIUFVzbksvR3B6azZNPQo9NllwZwotLS0tLUVORCBQR1AgU0lHTkFUVVJFLS0tLS0K",
                        "validated": false
                      }
                    }
                  ],
                  "signature": {
                    "pgp": "LS0tLS1CRUdJTiBQR1AgU0lHTkFUVVJFLS0tLS0KCndzRmNCQUFCQ0FBUUJRSm1lR2s5Q1JDMWFRN3V1NVVobEFBQUN2Z1FBR1drUG4xbXI2bGViaVlsRjJwZTRYY3gKVENQMkRQQVBnM2NnWExFQkt6dndlU0NCRXE1Zi9UaEVOT3lhOWlxYUFvQmRkM2Z6c1YrekdlTmpYRXFRTHI0dApuZkY1cE40THg4R2V2bzNyeXBESnlBd29kR2xFdEFEcjFrbEthSDFKSWZNdkFxaEExcmZMVElmQ01nTTNLVEF6CmczaHhOVXN0TnJoY2JETUZtYUVzaU5NeFRlTXBRSmlJcjBSRmJnVXcxWkhoU0NydzhISGxydjE5VC8wUWFXcnQKR1pjaHdzd2NOUHVEZm4vRjRCZlNEV2F2MjBUandMTmlxY2lndFgrQzBScVRqZkhVWnhjcSt4eWpCaTNnWHpTNQp0bGxEVzdneVJvR2pDdTAyZ2FiV3YwcnFtVUx1cHpHYTBYRUl6M0J5bDVwOXl3WFY3d1czdERjVkNLRU9YMzBnCnR2aWNkQ3B0Zjl2NlovZHRSNG1YM2lVeUdYRlZkM01xY2ptaGlqcEFJbW0yVXJORElVUVFRVkpkMzlGeE9uOVEKU29qd3N1b1FXYUhXbEdkWTdEWTVia2ZYM1cvNXNSZjlBVjdUemV5VjQvcmVnZ3EzSGY5ZC9QZEtaVEdLRXh3bwpUbXVNcTVUdmdVMWNjUTUzSkxrNVFGS0lzVmErZEh4SXd5YW5rcWNIbFBCRXlSSmtJajZlSVZRVldNQndCYUNhCkIvSnoxeGcweWc2SS9EZnRUQVdwTlBMUFQ2eFNBM09IZnZqQm1NblZGMzN0YnQ3Y3JtYVg0Wm5IbndZazZESVQKSUFNRkM4ay82OUZ6SFVRSWJJVTQrQk5CbVRpaTdnS2Y0cDlWNGRBNk95cTZFU1lkVlg2L2Z0OUNLMUY4eGxrQQpvSDNHMTI1dFdQQ0ZrTk9jb3ZGLwo9UnFzNwotLS0tLUVORCBQR1AgU0lHTkFUVVJFLS0tLS0KCg==",
                    "validated": false
                  }
                }
              }
            }
          ]
        ]
      }
    }
  }
}
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "predicateType": "docker.com/tape/OriginalImageRef/v0.1",
  "subject": [
    {
      "name": "kustomize/deployment.yaml",
      "digest": {
        "sha256": "5dcd7a6bd78c6d3613eefdea4747d2ba7e251ee355d793e165c9862ca7d69c9c"
      }
    }
  ],
  "predicate": {
    "foundImageReference": {
      "reference": "ghcr.io/stefanprodan/podinfo:6.7.0",
      "line": 26,
      "column": 16
    }
  }
}
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "predicateType": "docker.com/tape/ReplacedImageRef/v0.1",
  "subject": [
    {
      "name": "kustomize/deployment.yaml",
      "digest": {
        "sha256": "ad84dbd1450a752ae6ad1d99250310beca8fe7031da69f72a7a38fd95f02fe38"
      }
    }
  ],
  "predicate": {
    "replacedImageReference": {
      "reference": "ttl.sh/tape/podinfo:app.e1053e7cf3b59bec02980dd5cbbbb0cd151056e32810d25ee67ef52cb388a53f@sha256:d2b3cd93a48acdc91327533ce28fcb3169b2d9feaf73817dc2eb68858df64edb",
      "line": 26,
      "column": 16,
      "alias": "podinfo"
    }
  }
}
{
  "_type": "https://in-toto.io/Statement/v0.1",
  "predicateType": "docker.com/tape/ResolvedImageRef/v0.1",
  "subject": [
    {
      "name": "kustomize/deployment.yaml",
      "digest": {
        "sha256": "5dcd7a6bd78c6d3613eefdea4747d2ba7e251ee355d793e165c9862ca7d69c9c"
      }
    }
  ],
  "predicate": {
    "resolvedImageReference": {
      "reference": "ghcr.io/stefanprodan/podinfo:6.7.0@sha256:d2b3cd93a48acdc91327533ce28fcb3169b2d9feaf73817dc2eb68858df64edb",
      "line": 26,
      "column": 16,
      "alias": "podinfo"
    }
  }
}
$
```

## FAQ

### What configuration formats does Tape support, does it support any kind of templating?

Tape supports plain JSON and YAML manifest, which was the scope of the original experiment.
If the project was to continue, it could accommodate a variety of popular templating options,
e.g. CUE, Helm, and scripting languages, paving a way for a universal artifact format.

### How does Tape relate to existing tools?

Many existing tools in this space help with some aspects of handling Kubernetes resources. These tools operate on
either loosely coupled collections of resouces (like Kustomize), or opinionated application package formats (most
notably Helm). One of the goals of Tape is to abstract the use of any tools that already exist while paving the way
for innovation. Tape will attempt to integrate with most of the popular tools, and enable anyone to deploy applications
from taped images without having to know if under the hood it will use Kustomize, Helm, just plain manifest, or something
else entirely. The other goal is that users won't need to know about Tape either, perhaps someday `kubectl apply` could
support OCI artifacts and there could be different ways of building the artifacts.

### What kind of applications can Tape package?

Tape doesn't infer an opinion of how the application is structured, or what it consists of or doesn't consist of. It doesn't
present any application definition format, it operates on plain Kubernetes manifests found in a directory.

### Does Tape provide SBOMs?

Tape doesn't explicitly generate or process SBOMs, but fundamentally it could provide functionality around that.

## Acknowledgments & Prior Art

What Tape does is very much in the spirit of Docker images, but it extends the idea by shifting the perspective to configuration
as an entry point to a map of dependencies, as opposed to the forced separation of app images and configuration.

It's not a novelty to package configuration in OCI, there are many examples of this, yet that in itself doesn't provide for interoperability.
One could imagine something like Tape as a model that abstracts configuration tooling so that end-users don't need to think about whether
a particular app needs to be deployed with Helm, Kustomize, or something else.

Tape was directly inspired by [flux push artifact](https://fluxcd.io/flux/cheatsheets/oci-artifacts/). Incidentally, it also resembles
some of the aspects of CNAB, but it is much smaller in scope.

Tape was originally created in Docker Labs under [docker/labs-tape](https://github.com/docker/labs-tape), and it's now maintained by the original author as [errordeveloper/tape](https://github.com/errordeveloper/tape).
