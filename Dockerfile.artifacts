FROM goreleaser/goreleaser as builder
WORKDIR /build
COPY . /build
RUN goreleaser release --snapshot --skip-publish --rm-dist

FROM alpine
COPY --from=builder /build/dist/ /dist/
RUN mkdir -p /artifacts && \
    cp /dist/*.tar.gz /artifacts/ && \
    cp /dist/*.txt /artifacts/ && \
    ls -la /artifacts/*
CMD ["sleep", "150"]