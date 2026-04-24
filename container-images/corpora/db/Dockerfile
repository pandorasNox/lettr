FROM mariadb:12.2.2@sha256:e0236fc6386e7eacd9359e59d0a078bd7aa0d18280d36d13061121bedeaee903

RUN set -e; \
    test "1" = "$(cat /etc/os-release | grep -i 'Name="Ubuntu"' | wc -l)"; \
    test "1" = "$(cat /etc/os-release | grep 'VERSION="24.04.2 LTS (Noble Numbat)"' | wc -l)";

RUN set -e; \
    apt-get update; \
    apt-get install -y \
        curl \
    ;

RUN mkdir -p /scripts
COPY ./check_db_con.sh /scripts/check_db_con.sh
COPY ./import.sh /scripts/import.sh
COPY ./export.sh /scripts/export.sh

COPY ./datasets_download.sh /scripts/datasets_download.sh
RUN set -e; mkdir -p /datasets; /scripts/datasets_download.sh /datasets
