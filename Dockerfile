# syntax=docker/dockerfile:1.3-labs 
FROM ruby:3-alpine as base

LABEL maintainer="Mike Glenn <mglenn@ilude.com>"
LABEL ilude-project=joyride

RUN \
  mkdir -p /app /etc/hosts.d /etc/dnsmasq.d && \
  touch /etc/hosts.d/hosts && \
  touch /etc/dnsmasq.d/hosts && \
  echo 'install: --no-document' >> /etc/gemrc && \
  echo 'update: --no-document' >> /etc/gemrc && \ 
  apk add --no-cache \
    bash \
    dnsmasq \
    shadow \
    tzdata \
  && rm -rf \
    /usr/lib/ruby/gems/*/cache/* \
    /root/.gem/ \
    /var/cache/apk/* \
    /tmp/* \
    /var/tmp/*

ENV APP /app
ENV GEM_HOME /gems

WORKDIR $APP

CMD [ "./joyride" ] 

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

RUN wget https://releases.hashicorp.com/serf/0.8.2/serf_0.8.2_linux_amd64.zip -O /tmp/serf_0.8.2_linux_amd64.zip && \
    unzip /tmp/serf_0.8.2_linux_amd64.zip -d /usr/local/bin


##############################
# Begin production 
##############################
FROM base AS production

COPY --from=builder ${GEM_HOME} ${GEM_HOME}
COPY --from=builder /usr/local/bin/serf /usr/local/bin
COPY ./app $APP

