# Run arm64 virt VM (QEMU)

Locate your QEMU's shares directory in your system and set `QEMU_SHARES` accordingly 
```shell script
$ qemu-img create alpine.qcow 5G
$ QEMU_SHARES=/path/to/qemu/shares
$ qemu-system-aarch64 \
  -bios QEMU_SHARES/edk2-aarch64-code.fd \
  -cpu cortex-a72 -m 1024 -M virt,highmem=no \
  -cdrom path/to/alpine-standard-3.15.0-aarch64.iso \
  -boot d \
  -nic user,ipv6=off,model=e1000 \
  -smp 4 \
  -hda alpine.qcow \
  -rtc base=localtime \
  -nographic
```

Login as root:

    # login: root
    # password: (empty, hit enter)
    
Configure Alpine VM:

```shell script
uname -m

ip l set dev eth0 up
udhcpc eth0

ntpd -d -q -n -p uk.pool.ntp.org

cat > /etc/apk/repositories << EOF; $(echo)
http://dl-cdn.alpinelinux.org/alpine/v$(cat /etc/alpine-release | cut -d'.' -f1,2)/main/
http://dl-cdn.alpinelinux.org/alpine/v$(cat /etc/alpine-release | cut -d'.' -f1,2)/community/
http://dl-cdn.alpinelinux.org/alpine/edge/testing/
EOF

apk update
apk add podman file

mount -t tmpfs cgroup_root /sys/fs/cgroup
mkdir /sys/fs/cgroup/blkio
mount -t cgroup -o blkio none /sys/fs/cgroup/blkio
```

Do your testing:

````shell script
podman pull quay.io/bugfest/tor-controller:latest
podman image inspect quay.io/bugfest/tor-controller:latest
podman run --rm -ti quay.io/bugfest/tor-controller:latest --help

file /var/lib/containers/storage/overlay/c450f82aba630e856a394ce33b4b09f02db5522aa930f1f163b8e9c8e02146f7/diff/manager
# /var/lib/containers/storage/overlay/c450f82aba630e856a394ce33b4b09f02db5522aa930f1f163b8e9c8e02146f7/diff/manager: ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, Go BuildID=1fZSGYRhNaI79hhbTcgx/4Ek_ZM6bIpAdo2fuYCcf/maZ7Wemhq3FVhv5pbZFB/fgXWOJPsDeddc3FrBWWB, not stripped
````

(optional) persistent installation
```shell script
setup-alpine
setup-ntp -c chrony

vi /etc/default/grub
# GRUB_CMDLINE_LINUX_DEFAULT="modules=sd-mod,usb-storage,ext4 quiet rootfstype=ext4 cgroup_memory=1 cgroup_enable=memory cgroup_enable=cpuset"

grub-mkconfig > /boot/grub/grub.cfg

poweroff
```

Boot from disk
```shell script
$ qemu-system-aarch64 \
  -bios QEMU_SHARES/edk2-aarch64-code.fd \
  -cpu cortex-a72 -m 1024 -M virt,highmem=no \
  -boot c \
  -nic user,ipv6=off,model=e1000 \
  -smp 4 \
  -hda alpine.qcow \
  -rtc base=localtime \
  -nographic
```

Install k3s
```shell script
apk add curl
curl -sfL https://get.k3s.io | sh -

apk add bash
curl -k https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

cp /etc/rancher/k3s/k3s.yaml /root/.kube/config
helm repo add bugfest https://bugfest.github.io/tor-controller
helm upgrade --install tor-controller bugfest/tor-controller

kubectl -n default get po -l app.kubernetes.io/name=tor-controller
kubectl apply -f https://raw.githubusercontent.com/bugfest/tor-controller/master/hack/sample/echoserver.yaml
kubectl apply -f https://raw.githubusercontent.com/bugfest/tor-controller/master/hack/sample/onionservice.yaml
kubectl get onionservice/example-onion-service -o template='{{printf "%s\n" .status.hostname}}'

```

Docs
----

- qemu docs: https://qemu.readthedocs.io/en/latest/system/invocation.html#hxtool-1
- alpine downloads: https://alpinelinux.org/downloads/
- determine bin architecture: https://exceptionshub.com/determine-target-architecture-of-binary-file-in-linux-li  ary-or-executable.html
- fix blkio cgroup issue: https://www.programmerall.com/article/5933169238/
- fix alpine repos: https://github.com/alpinelinux/docker-alpine/issues/98
- fix alpine time: https://wiki.alpinelinux.org/wiki/Alpine_Linux:FAQ#Time_and_timezones
