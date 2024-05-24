FROM ghcr.io/pulumi/devcontainer:extra

##################################################################################
# Install AWS CLI v2

USER root

RUN 

RUN echo \
    && export ARCH=$(uname -m | awk '{ if ($1 == "x86_64") print "x86_64"; else if ($1 == "aarch64" || $1 == "arm64") print "aarch64"; else print "unknown" }') \
    && export NAME="aws" \
    && export TEST="${NAME} --version" \
    && export PKG="awscli-exe-linux-${ARCH}.zip" \
    && export URL="https://awscli.amazonaws.com/${PKG}" \
    && export DIR="/tmp/awscli" \
    && echo "---------------------------------------------------------"\
    && echo "INFO[${NAME}] Installed:" \
    && echo "INFO[${NAME}]   Command:        ${NAME}" \
    && echo "INFO[${NAME}]   Package:        ${PKG}" \
    && echo "INFO[${NAME}]   Architecture:   ${ARCH}" \
    && echo "INFO[${NAME}]   Source:         ${URL}" \
    && echo "---------------------------------------------------------"\
    && mkdir -p ${DIR} \
    && ${curl} ${URL} --output ${DIR}/${PKG} \
    && unzip ${DIR}/${PKG} -d ${DIR} \
    && bash -c "${DIR}/aws/install" \
    && ${dir_clean} \
    && ${TEST} \
    && echo


# install tofu
RUN echo \
    && export TOFU_URL=https://get.opentofu.org/install-opentofu.sh \
    && export TOFU_SCRIPT=install-opentofu.sh \
    && curl --proto '=https' --tlsv1.2 -fsSL  ${TOFU_URL} -o ${TOFU_SCRIPT} \
    && chmod +x ${TOFU_SCRIPT} \
    && ./${TOFU_SCRIPT} --install-method deb \
    && rm -f ${TOFU_SCRIPT}


# install coder
RUN echo \
    && export CODER_URL=https://coder.com/install.sh \
    && curl -L ${CODER_URL} | sh

USER vscode