ARG BUILD_ENV=dapper
ARG ARCH

FROM registry.suse.com/bci/bci-base:15.5
RUN zypper -n update && \
    zypper -n install openssh catatonit git-core && \
    zypper -n clean -a
RUN useradd -u 1000 -U -m gitjob

FROM base AS copy_dapper
ONBUILD ARG ARCH
ONBUILD COPY bin/gitjob-linux-${ARCH} /usr/bin/gitjob
ONBUILD COPY bin/gitcloner-linux-${ARCH} /usr/bin/gitcloner

FROM base AS copy_goreleaser
ONBUILD ARG ARCH
ONBUILD COPY gitjob-linux-${ARCH} /usr/bin/gitjob
ONBUILD COPY gitcloner-linux-${ARCH} /usr/bin/gitcloner

USER 1000
ENTRYPOINT ["catatonit", "--"]
CMD ["gitjob"]
