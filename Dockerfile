FROM ubuntu:latest
MAINTAINER Christoph Coijanovic <hi@coijanovic.com>

ARG NYM_VERSION=nym-binaries-v1.1.1
ARG GO_VERSION=1.19.3

##############
# Building Nym
# See: https://nymtech.net/docs/stable/run-nym-nodes/build-nym
##############

# Install basics
RUN apt update
RUN apt install -y curl git python3 python3-pip 

# Install Rust
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH="/root/.cargo/bin:${PATH}"

# Install other Prerequisites
RUN apt install -y pkg-config build-essential libssl-dev curl jq git 

# Clone Repo and Build
RUN rustup update
RUN git clone https://github.com/nymtech/nym.git
WORKDIR nym
RUN git checkout $NYM_VERSION
RUN cargo build --release

#################
# Building Panini
#################

# Install go
WORKDIR /tmp
RUN curl -OL "https://golang.org/dl/go${GO_VERSION}.linux-amd64.tar.gz"
RUN rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go${GO_VERSION}.linux-amd64.tar.gz
ENV PATH="${PATH}:/usr/local/go/bin"
RUN go version

WORKDIR /panini
COPY go.mod ./
COPY go.sum ./

COPY *.go ./
COPY sender/*.go ./sender/
COPY receiver/*.go ./receiver/
COPY message/*.go ./message/
COPY utils/*.go ./utils/

RUN go mod download

RUN go build -o /runpanini main.go

COPY ./run.sh /
RUN chmod +x /run.sh
