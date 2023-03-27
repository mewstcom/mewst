FROM heroku/builder:22

USER root

RUN apt-get update
RUN apt-get install -y --no-install-recommends \
  liblzma-dev \
  libpq-dev \
  libxml2-dev \
  libxslt1-dev \
  patch \
  zlib1g-dev
RUN apt-get clean
RUN rm -rf /var/lib/apt/lists/*

USER cnb
