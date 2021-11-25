ARG GO_VERSION_ARG=1.17
ARG NGINX_VERSION_ARG=1.20.1
ARG MARIADB_VERSION_ARG=10.6.4
ARG ALPHINE_VERSION_ARG=3.10

FROM bitnami/git:2 AS git
RUN mkdir -p /opt/docker-library
WORKDIR /opt/docker-library
RUN git clone https://github.com/docker-library/healthcheck.git /opt/docker-library/healthcheck


FROM golang:${GO_VERSION_ARG} AS app_builder
ARG GO_VERSION_ARG
COPY ./.env /var/www/app/.env
COPY ./go /var/www/app
WORKDIR /var/www/app
RUN unset GOPATH \
    && go mod tidy \
    && go mod edit -go ${GO_VERSION_ARG} \
    && go mod download \
    && go build -o app ./cmd/

FROM alpine:${ALPHINE_VERSION_ARG} AS app_runner
RUN apk add --no-cache \
        libc6-compat
COPY ./docker/app/docker-command.sh /bin/docker-command.sh
COPY ./go/templates /var/www/app/templates
COPY --from=app_builder /var/www/app/app /var/www/app/app
WORKDIR /var/www/app
CMD ./app

FROM nginx:${NGINX_VERSION_ARG} AS web
ARG BASIC_AUTH_ARG
COPY ./docker/web/docker-command.sh /bin/docker-command.sh
RUN touch /etc/nginx/.htpasswd \
    && echo "${BASIC_AUTH_ARG}" > /etc/nginx/.htpasswd
RUN apt-get update && apt-get install -y nginx-extras
RUN chmod +x /bin/docker-command.sh
CMD ["/bin/docker-command.sh"]


FROM mariadb:${MARIADB_VERSION_ARG} AS mariadb
RUN mkdir -p /etc/mysql/conf.d
ARG TZ_ARG="Europe/Moscow"
RUN ln -snf /usr/share/zoneinfo/${TZ_ARG} /etc/localtime && echo ${TZ_ARG} > /etc/timezone
COPY ./docker/mariadb/config/my.cnf /etc/mysql/conf.d/my.cnf
COPY --from=git /opt/docker-library/healthcheck/mysql/docker-healthcheck /usr/local/bin/
RUN chmod +x /usr/local/bin/docker-healthcheck
RUN chmod 0444 /etc/mysql/conf.d/my.cnf \
    && mkdir -p /var/log/mysql \
    && touch /var/log/mysql/mysql-slow.log \
    && chown -R mysql:mysql /var/log/mysql
VOLUME ["/var/log/mysql"]
HEALTHCHECK CMD ["docker-healthcheck"]
