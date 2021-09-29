FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY keycloak-attribute-sync-controller .
USER 65532:65532

ENTRYPOINT ["/keycloak-attribute-sync-controller"]
