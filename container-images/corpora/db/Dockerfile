FROM mariadb:11.8.3@sha256:272084c2dec70619714df329c4ffcb336e3f8c723072c3f56f2e4015997bbf2c

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
