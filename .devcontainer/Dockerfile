FROM ghcr.io/lesomnus/dev-golang:1.25

ARG TARGETARCH
ARG TARGETVARIANT


RUN VERSION="1.3.0" \
	TEMP=$(mktemp -d); \
    cd "${TEMP}" \
    && curl -o a.tar.gz -fsSL "https://github.com/oras-project/oras/releases/download/v${VERSION}/oras_${VERSION}_linux_${TARGETARCH}${TARGETVARIANT}.tar.gz" \
	&& tar -xzf a.tar.gz \
	&& mv oras /usr/local/bin/oras \
	&& rm -rf "${TEMP}"

RUN mkdir -p /go/pkg/mod \
	&& chown -R 1000:1000 /go



USER hypnos

RUN mkdir -p /home/hypnos/.cache/go-build

# Comment out the following lines if you don't want to use Zsh.
RUN wget https://github.com/robbyrussell/oh-my-zsh/raw/master/tools/install.sh -O - | zsh || true \
	&& git clone https://github.com/zsh-users/zsh-autosuggestions         ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-autosuggestions     \
	&& git clone https://github.com/zsh-users/zsh-syntax-highlighting.git ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-syntax-highlighting \
	&& git clone https://github.com/zsh-users/zsh-completions             ${ZSH_CUSTOM:-~/.oh-my-zsh/custom}/plugins/zsh-completions
ADD .zshrc /home/hypnos
ENV SHELL=/bin/zsh
