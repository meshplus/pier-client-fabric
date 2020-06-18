FROM golang:1.14.2 as builder

RUN mkdir -p /go/src/github.com/meshplus/pier
RUN mkdir -p /go/src/github.com/meshplus/pier-client-fabric
WORKDIR /go/src/github.com/meshplus/pier

# Cache dependencies
COPY go.mod ../pier-client-fabric/
COPY go.sum ../pier-client-fabric/
COPY build/pier/go.mod .
COPY build/pier/go.sum .

RUN go env -w GOPROXY=https://goproxy.cn,direct
RUN go mod download -x

#RUN apk add make

# Build real binaries
COPY build/pier .
COPY . ../pier-client-fabric/

RUN go get -u github.com/gobuffalo/packr/packr

RUN make install

RUN cd ../pier-client-fabric && \
    make fabric1.4 && \
    cp build/fabric-client-1.4.so /go/bin/fabric-client-1.4.so

# Final image
FROM frolvlad/alpine-glibc

WORKDIR /root

# Copy over binaries from the builder
COPY --from=builder /go/bin/pier /usr/local/bin
COPY ./build/pier/build/libwasmer.so /lib
ENV LD_LIBRARY_PATH=$LD_LIBRARY_PATH:/lib

RUN ["pier", "init"]

RUN mkdir -p /root/.pier/plugins
COPY --from=builder /go/bin/*.so /root/.pier/plugins/
COPY config/validating.wasm /root/.pier/validating.wasm
COPY scripts/docker_entrypoint.sh /root/docker_entrypoint.sh
RUN chmod +x /root/docker_entrypoint.sh

COPY config /root/.pier/fabric
COPY config/pier.toml /root/.pier/pier.toml

ENV APPCHAIN_NAME=fabric

EXPOSE 44555 44544

ENTRYPOINT ["/root/docker_entrypoint.sh", "$APPCHAIN_NAME"]
