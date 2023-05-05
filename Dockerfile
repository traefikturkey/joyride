# syntax=docker/dockerfile:1.3-labs 
FROM ruby:3-alpine as base

LABEL maintainer="Mike Glenn <mglenn@ilude.com>"
LABEL ilude-project=joyride

RUN \
  mkdir -p /app /usr/local/etc \
  && { \
  echo 'install: --no-document'; \
  echo 'update: --no-document'; \
  } >> /usr/local/etc/gemrc \
  && apk add --no-cache bash shadow tzdata \
  && rm -rf \
    /usr/lib/ruby/gems/*/cache/* \
    /root/.gem/ \
    /var/cache/apk/* \
    /tmp/* \
    /var/tmp/*

ARG USER=anvil
ENV USER=${USER}
ARG PUID=1000
ARG PGID=1000
ENV APP /app
ENV GEM_HOME /gems

RUN \
  groupadd -g $PGID $USER && \
  useradd -s /sbin/nologin -g $PGID -u $PUID -d /home/$USER $USER && \
  mkdir -p /home/anvil && \
  chown $USER:$USER $APP

WORKDIR $APP

COPY --chmod=755 <<-"EOF" /usr/local/bin/docker-entrypoint.sh
#!/bin/sh
set -e

exec $@
EOF

EXPOSE 54/udp

ENTRYPOINT ["docker-entrypoint.sh"]
CMD [ "bundle", "exec", "ruby", "joyride.rb" ]

##############################
# Begin builder
##############################
FROM base AS builder

RUN apk --no-cache add \
  build-base \
  git

COPY ./app/Gemfile $APP
#COPY ./app/Gemfile.lock $APP
RUN bundle install

##############################
# Begin production 
##############################
FROM base AS production

COPY --from=builder ${GEM_HOME} ${GEM_HOME}
COPY ./app $APP

USER $USER