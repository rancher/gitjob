FROM oraclelinux:7-slim
RUN yum update -y && yum install -y git openssh && adduser -u 1000 gituser
COPY bin/gitjob /usr/bin/
USER 1000
CMD ["gitjob"]
