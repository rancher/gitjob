ARG BUILD_ENV=docker
ARG ARCH

FROM registry.suse.com/bci/bci-base:15.6 AS base
RUN zypper -n update && \
    zypper -n install git openssh catatonit && \
    zypper -n clean -a
RUN useradd -u 1000 -U -m gitjob

FROM base AS copy_docker
ONBUILD ARG ARCH
ONBUILD COPY bin/gitjob /usr/bin/gitjob

FROM base AS copy_buildx
ONBUILD ARG TARGETARCH
ONBUILD COPY bin/gitjob-linux-$TARGETARCH /usr/bin/gitjob

FROM base AS copy_goreleaser
ONBUILD ARG ARCH
ONBUILD COPY gitjob-linux-${ARCH} /usr/bin/gitjob

FROM copy_${BUILD_ENV}
USER 1000
ENTRYPOINT ["catatonit", "--", "gitjob"]
