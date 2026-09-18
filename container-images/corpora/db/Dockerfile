FROM mariadb:12.3.3@sha256:805c8e104bd563d5bfa24fadd3f31cd419ea859cb5277f32b5dbf2db714f9ed1

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
