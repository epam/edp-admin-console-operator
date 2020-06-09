FROM alpine:3.11.3

ENV OPERATOR=/usr/local/bin/admin-console-operator \
    USER_UID=1001 \
    USER_NAME=admin-console-operator \
    HOME=/home/admin-console-operator

RUN apk add --no-cache ca-certificates openssh-client

# install operator binary
COPY admin-console-operator ${OPERATOR}

COPY build/bin /usr/local/bin
COPY build/configs /usr/local/configs

RUN  chmod u+x /usr/local/bin/user_setup && \
     chmod ugo+x /usr/local/bin/entrypoint && \
     /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}