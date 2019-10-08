FROM scratch

COPY build/_output/bin/multicloud-operators-placementrule .
CMD ["./multicloud-operators-placementrule"]
