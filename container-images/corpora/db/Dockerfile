FROM mariadb:11.3.2@sha256:e101f9db31916a5d4d7d594dd0dd092fb23ab4f499f1d7a7425d1afd4162c4bc

RUN set -e; \
    test "1" = "$(cat /etc/os-release | grep -i 'Name="Ubuntu"' | wc -l)"; \
    test "1" = "$(cat /etc/os-release | grep 'VERSION="22.04.4 LTS (Jammy Jellyfish)"' | wc -l)";

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
