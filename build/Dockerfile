FROM scratch

ENV OPERATOR=/usr/local/bin/multicloud-operators-placementrule \
    USER_UID=1001 \
    USER_NAME=multicloud-operators-placementrule

# install operator binary
COPY build/_output/bin/multicloud-operators-placementrule ${OPERATOR}

COPY build/bin /usr/local/bin
RUN  /usr/local/bin/user_setup

ENTRYPOINT ["/usr/local/bin/entrypoint"]

USER ${USER_UID}
