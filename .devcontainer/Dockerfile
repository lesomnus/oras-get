FROM ghcr.io/lesomnus/dev-golang:1.26

ARG TARGETARCH
ARG TARGETVARIANT

RUN VERSION="1.3.2" \
	TEMP=$(mktemp -d); \
    cd "${TEMP}" \
    && curl -o a.tar.gz -fsSL "https://github.com/oras-project/oras/releases/download/v${VERSION}/oras_${VERSION}_linux_${TARGETARCH}${TARGETVARIANT}.tar.gz" \
	&& tar -xzf a.tar.gz \
	&& mv oras /usr/local/bin/oras \
	&& rm -rf "${TEMP}"
