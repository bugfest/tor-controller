# Run arm64 virt VM (QEMU)

Locate your QEMU's shares directory in your system and set `QEMU_SHARES` accordingly 
```shell script
$ QEMU_SHARES=/path/to/qemu/shares
$ qemu-system-aarch64 \
  -bios QEMU_SHARES/edk2-aarch64-code.fd \
  -cpu cortex-a72 -m 1024 -M virt,highmem=no \
  -cdrom path/to/alpine-standard-3.15.0-aarch64.iso \
  -boot d \
  -nic user,ipv6=off,model=e1000 \
  -smp 4 \
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

Docs
----

- qemu docs: https://qemu.readthedocs.io/en/latest/system/invocation.html#hxtool-1
- alpine downloads: https://alpinelinux.org/downloads/
- determine bin architecture: https://exceptionshub.com/determine-target-architecture-of-binary-file-in-linux-library-or-executable.html
- fix blkio cgroup issue: https://www.programmerall.com/article/5933169238/
- fix alpine repos: https://github.com/alpinelinux/docker-alpine/issues/98
- fix alpine time: https://wiki.alpinelinux.org/wiki/Alpine_Linux:FAQ#Time_and_timezones
