For a traditional networking lab orchestration system `containerlab` appears to be quite unique in a way that it runs containers, not VMs. This inherently means that container images need to be available to spin up the nodes.

To keep things simple, containerlab adheres to the same principles of referencing container images as common tools like docker, podman, k8s do. The following example shows a clab file that references container images using various forms:

```yaml
name: images
topology:
  nodes:
    node1:
      # image from docker hub registry with implicit `latest` tag
      image: alpine
    node2:
      # image from docker hub with explicit tag
      image: ubuntu:20.04
    node3:
      # image from github registry
      image: ghcr.io/hellt/network-multitool
    node4:
      # image from some private registry
      image: myregistry.local/private/alpine:custom
```

When containerlab launches a lab, it reads the image name from the topology file and expects to find the referenced images locally or by pulling them from the registry.

If in the example above, the image named `myregistry.local/private/alpine:custom` was not loaded to docker local image store before, containerlab will attempt to pull this image and will expect the private registry to be reachable.

Container images offer a great flexibility and reproducibility of lab builds, to embrace it fully, we wanted to capture some basic image management operations and workflows in this article.

## Managed images

Containerlab can manage image builds as part of a lab deployment. Managed images are declared under the top-level `images` section of the topology file, next to `name`, `mgmt`, `settings` and `topology`.

This keeps node definitions focused on runtime intent. A node still starts from a single `image`; the `images` section describes how selected images are produced before they are used by nodes.

```yaml
name: managed-images

images:
  dpu-base:
    image: localhost/dpu-base:latest
    build:
      mode: pre-deploy
      context: ./dpu-base
      rebuild: if-missing

  dpu-01:
    image: localhost/dpu-01:stage3
    build:
      mode: topology
      node: dpu-01
      rebuild: if-missing
      builder:
        image: localhost/dpu-base:latest
        cmd: /usr/local/bin/build-dpu-stage3
      commit:
        cmd: ["/usr/sbin/init"]

topology:
  nodes:
    dpu-01:
      kind: linux
      image: localhost/dpu-01:stage3
```

In this example, `dpu-base` is built before any node is deployed. The `dpu-01` node starts from `localhost/dpu-01:stage3`, and containerlab knows that this image should be produced by the `dpu-01` managed image target before the final node is started.

The key under `images`, such as `dpu-base` or `dpu-01`, is a logical name used by the topology author. Nodes consume managed images by referencing the resulting image tag in their `image` field.

### Build modes

The `build.mode` field defines when and how an image is built.

| Mode | Description |
| ---- | ----------- |
| `pre-deploy` | Builds the image with the container runtime image build API before node images are pulled and before nodes are created. |
| `topology` | Starts a temporary builder container in the selected node's topology position, attaches its links, runs a build handoff command, commits the stopped container as the target image, removes the temporary container, and then starts the final node from the committed image. |

If `mode` is omitted, `pre-deploy` is used.

### Rebuild policy

The `build.rebuild` field controls whether containerlab should rebuild an image when the output tag already exists locally.

| Value | Description |
| ----- | ----------- |
| `if-missing` | Build the image only when the output image is missing. This is the default. |
| `always` | Build the image on every deployment. |
| `never` | Never build the image. Fail if the output image is missing. |

### Pre-deploy builds

`pre-deploy` builds use Dockerfile-style image builds. Paths are resolved relative to the topology file when they are not absolute.

```yaml
images:
  tools:
    image: localhost/tools:latest
    build:
      mode: pre-deploy
      context: ./tools
      dockerfile: Dockerfile
      network: host
      rebuild: if-missing
```

The supported fields are:

| Field | Description |
| ----- | ----------- |
| `context` | Build context path. Defaults to `.`. |
| `dockerfile` | Dockerfile name or path relative to the build context. Defaults to `Dockerfile`. |
| `network` | Network mode passed to the image build operation. |
| `rebuild` | Rebuild policy. |

### Topology builds

`topology` builds are useful when an image must be produced from a container that has been placed into the lab topology. A common pattern is to start from a reusable base image, let a topology-connected builder command discover or prepare node-specific state, then commit the resulting container as the image that the final node will run.

```yaml
images:
  dpu-01:
    image: localhost/dpu-01:stage3
    build:
      mode: topology
      node: dpu-01
      rebuild: if-missing
      builder:
        image: localhost/dpu-base:latest
        cmd: /usr/local/bin/build-dpu-stage3
      commit:
        entrypoint: []
        cmd: ["/usr/sbin/init"]

topology:
  nodes:
    dpu-01:
      kind: linux
      image: localhost/dpu-01:stage3
```

For topology builds, `build.builder.image` and `build.builder.cmd` are required.

The `node` field selects the topology node whose placement and links are used for the temporary builder container. If `node` is omitted, containerlab can infer it only when exactly one node consumes the target image. If multiple nodes reference the same target image, the build is ambiguous and containerlab will ask you to set `build.node`.

The supported topology build fields are:

| Field | Description |
| ----- | ----------- |
| `node` | Name of the node whose topology position is used for the temporary builder container. |
| `builder.image` | Image used to start the temporary builder container. Required for `topology` mode. |
| `builder.cmd` | Command executed inside the temporary builder container. Required for `topology` mode. |
| `builder.entrypoint` | Optional entrypoint override for the temporary builder container. |
| `commit.entrypoint` | Entrypoint stored in the committed image. |
| `commit.cmd` | Command stored in the committed image. |
| `rebuild` | Rebuild policy. |

### Sharing build stages

Managed image targets are reusable by image tag. This allows a shared base image to be built once and consumed by many later image builds.

```yaml
name: dpu-builds

images:
  dpu-base:
    image: localhost/dpu-base:latest
    build:
      mode: pre-deploy
      context: ./dpu-base
      rebuild: if-missing

  dpu-01:
    image: localhost/dpu-01:stage3
    build:
      mode: topology
      node: dpu-01
      builder:
        image: localhost/dpu-base:latest
        cmd: /usr/local/bin/build-dpu-stage3

  dpu-02:
    image: localhost/dpu-02:stage3
    build:
      mode: topology
      node: dpu-02
      builder:
        image: localhost/dpu-base:latest
        cmd: /usr/local/bin/build-dpu-stage3

topology:
  nodes:
    dpu-01:
      kind: linux
      image: localhost/dpu-01:stage3
    dpu-02:
      kind: linux
      image: localhost/dpu-02:stage3
```

The `dpu-base` image is the shared build input. The two topology builds produce distinct output images, because each target has a unique `image` value.

Containerlab rejects duplicate managed image targets that produce the same output image with different build definitions. Duplicate targets with identical build definitions are treated as the same target.

### Node-level build shorthand

When the build definition belongs to a single node and does not need a logical name, you can place `build` directly on the node. The node must still declare or inherit an `image`; internally, containerlab treats the node-level build as an anonymous managed image target for that image.

```yaml
topology:
  nodes:
    dpu-02:
      kind: linux
      image: localhost/dpu-02:stage3
      build:
        mode: topology
        rebuild: if-missing
        builder:
          image: localhost/dpu-base:latest
          cmd: /usr/local/bin/build-dpu-stage3
        commit:
          cmd: ["/usr/sbin/init"]
```

In this shorthand form, `dpu-02` still has a single runtime `image`. The nested `build` block tells containerlab how to create that image for this node before the final container is started.

The shorthand is convenient for one-off images. Prefer top-level `images` when a build target is shared, when the topology has several image stages, or when you want the topology file to make the image lifecycle explicit.

### Runtime support

Image build and commit operations require Docker runtime support. Podman does not support managed image builds in this release.

## Tagging images

A container image name can appear in various forms. A short form of `alpine` will be expanded by docker daemon to `docker.io/alpine:latest`. At the same time an image named `myregistry.local/private/alpine:custom` is already a fully qualified name and indicates the container registry (`myregistry.local`) image repository name (`private/alpine`) and its tag (`custom`).

With a [`docker tag`](https://docs.docker.com/engine/reference/commandline/tag/) command it is possible to "rename" an image to something else. This can be needed for various purposes, but most common needs are:

1. rename the image so it can be pushed to another repository
2. rename the image to users liking

Let's imagine that we have a private repository from which we pulled the image with a name `registry.srlinux.dev/pub/vr-sros:20.10.R3`. By using this name in our clab file we can make use of this image in our lab. But that is quite a lengthy name, we might want to shorten it to something less verbose:

```bash
# docker tag <old-name> <new-name>
docker tag registry.srlinux.dev/pub/vr-sros:20.10.R3 sros:20.10.R3
```

With that we make a new image named `sros:20.10.R3` that references the same original image. Now we can use the short name in our clab files.

### Pushing to a new registry

That same `docker tag` command can be used to rename the image so it can be pushed to another registry. For example consider the newly built SR OS 21.2.R1 [vrnetlab](vrnetlab.md) image that by default will have a name of `vrnetlab/vr-sros:21.2.R1`. This container image can't be pushed anywhere in its current form, but retagging will help us out.

If we wanted to push this image to a public registry like the Github Container Registry, we could do the following:

```bash
# retag the image to a fully qualified name that is suitable for
# push to github container registry
sudo docker tag vrnetlab/vr-sros:21.2.R1 ghcr.io/srl-labs/vr-sros:21.2.R1

# and now we can push it
sudo docker push ghcr.io/srl-labs/vr-sros:21.2.R1
```

## Exchanging images

Container images are a perfect fit for sharing. Once anyone built an image with a certain NOS inside it can share it with anyone via container registry. Sensitive and proprietary images are typically pushed to private registries and internal users pull it from there.

But sometimes you need to share an image with a colleague or your own setup that doesn't have access to a private registry. There are couple of ways to achieve that.

### As zipped tar archive

A container image can be saved as `tar.gz` file that you can then share via various channels:

```
sudo docker save vrnetlab/vr-sros:21.2.R1 | xz -T 0 > sros.tar.gz
```

Now you can push the tar.gz file to Google Drive, Dropbox, etc.

On the receiving end you can load the container image:

```
sudo docker load -i sros.tar.gz
```

### Via temp registry

Another cool way of sharing a container image is via [ttl.sh](https://ttl.sh) registry which offers a way to push an image to their public registry but the image will expire with a timeout you set.

For example, let's push our image to the ttl.sh registry under a random name and make it expire in 15 minutes.

```bash
# generate random 6 char sequence
IMAGE=$(cat /dev/urandom | tr -dc 'a-z0-9' | fold -w 6 | head -n 1)
# set ttl
TTL=15m

# tag and push
sudo docker tag vrnetlab/vr-sros:21.2.R1 ttl.sh/$IMAGE:$TTL
sudo docker push ttl.sh/$IMAGE:$TTL
echo "pull the image with \"docker pull ttl.sh/$IMAGE:$TTL\" in the next $TTL"
```

That is a very convenient way of sharing images with a small security compromise.
