FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY keycloak-attribute-sync-controller manager
USER 65532:65532

ENTRYPOINT ["/manager"]
