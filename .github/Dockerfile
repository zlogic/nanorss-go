FROM scratch

COPY --chown=1001:1001 nanorss-go  /usr/local/nanorss/
COPY --from=registry.access.redhat.com/ubi8/ubi-minimal \
    /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem \
    /etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem

WORKDIR /usr/local/nanorss

EXPOSE 8080
USER 1001
CMD [ "./nanorss-go" ]
