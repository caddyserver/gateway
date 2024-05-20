# Example

These examples rely on both [`cert-manager`](https://cert-manager.io/docs/installation/) and
[`cert-manager-csi-driver`](https://cert-manager.io/docs/usage/csi-driver/installation/). Please
ensure both are installed before proceeding.

In the future, more examples and options for handling the issuance and verification of mTLS
certificates between the Gateway and Caddy will be provided.

## Configuring TLS

This step is used to configure TLS for HTTPRoute and TLSRoute resources. These certificates will be
used to provide public-facing TLS certificates. While this step is optional, it is strongly
recommended.

This step shows you how to use `cert-manager` and an ACME provider to configure TLS for a Gateway.
If you already have your own certificate secret on your cluster, or want to use internally issued
certificates, you don't need to follow this guide.

You will need to ensure `cert-manager` has Gateway API support enabled. Please read
<https://cert-manager.io/docs/configuration/acme/http01/#configuring-the-http-01-gateway-api-solver>
for more information.

### DNS-01

If you would like to use wildcard certificates or issue trusted certificates without exposing the
Gateway to the public internet, you _must_ use an ACME issuer that supports DNS-01 challenges. For
more information on supported DNS providers, please read
<https://cert-manager.io/docs/configuration/acme/dns01/#supported-dns01-providers> for more
information.

Here is an example issuer for Cloudflare:

```yaml
---
apiVersion: v1
kind: Secret
metadata:
  name: cloudflare-api-token-secret
  namespace: cert-manager
type: Opaque
stringData:
  # https://cert-manager.io/docs/configuration/acme/dns01/cloudflare/#api-tokens
  api-token: ""
---
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: letsencrypt-dns
spec:
  acme:
    server: https://acme-v02.api.letsencrypt.org/directory
    privateKeySecretRef:
      name: letsencrypt-issuer-account-key
    solvers:
      - dns01:
          cloudflare:
            apiTokenSecretRef:
              name: cloudflare-api-token-secret
              key: api-token
```

### HTTP-01

If you don't need DNS challenges you can install our default LetsEncrypt ClusterIssuer without any
additional configuration.

```bash
kubectl apply -f https://raw.githubusercontent.com/caddyserver/gateway/master/example/letsencrypt.yaml
```

## Installation and Configuration

### Internal TLS

We require certificates to secure the communication between the Controller (this project) and Caddy.
Without mTLS between these two components, anyone could access the Caddy Admin API or potentially
view certificates referenced by Gateway resources in plain-text.

The default resources assume that you have a `Issuer` configured named `caddy`. This issuer must not
be a `SelfSigned` issuer and must be a `CA` or similar (like `Vault`). Do not use a public `ACME`
issuer such as LetsEncrypt as it won't work.

You are more than welcome to bring your own cert-manager issuer. Just update the
`csi.cert-manager.io/issuer-kind` and `csi.cert-manager.io/issuer-name` volume attributes for the
operator and Caddy, then skip the following `kubectl apply` command and move on to the
`Deploy the Operator` step.

This example creates both a SelfSigned issuer that bootstraps a regular CA issuer. If you are fine
with the default settings, then no changes need to be made.

```bash
kubectl apply -f https://raw.githubusercontent.com/caddyserver/gateway/master/example/internal-issuer.yaml
```

### Deploy the Operator

This deploys the Caddy Gateway Controller (the code in this repository). This is required in order
to program the actual Caddy web-server instances.

```bash
kubectl apply -f https://raw.githubusercontent.com/caddyserver/gateway/master/example/operator.yaml
```

### Deploy a Caddy instance

This deploys a Deployment of three Caddy instances, alongside a Load Balancer.

```bash
kubectl apply -f https://raw.githubusercontent.com/caddyserver/gateway/master/example/caddy.yaml
```

## Create the Gateway

After the operator is installed, you will need to create the actual Gateway that will utilize the
Caddy instance you just deployed.

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: Gateway
metadata:
  name: caddy
  namespace: caddy-system
  labels:
    app.kubernetes.io/name: caddy
    app.kubernetes.io/component: caddy
    app.kubernetes.io/instance: caddy
    app.kubernetes.io/part-of: caddy
  annotations:
    # These annotations are all optional if you don't want to use cert-manager to issue certificates.
    # Please ensure you set the `cert-manager.io/cluster-issuer` or `cert-manager.io/issuer` to the
    # correct issuer for your configuration.
    cert-manager.io/cluster-issuer: letencrypt
    cert-manager.io/private-key-rotation-policy: Always
    cert-manager.io/usages: digital signature, server auth
spec:
  gatewayClassName: caddy
  listeners:
    - name: http
      protocol: HTTP
      port: 80
      allowedRoutes:
        namespaces:
          from: All
    - name: https
      protocol: HTTPS
      port: 443
      allowedRoutes:
        namespaces:
          from: All
      # In-order to issue certificates or match requests, the Gateway API spec requires you specify
      # a hostname here. This is a placeholder you *must* replace if you want to use HTTPS.
      hostname: "*.example.com"
      tls:
        mode: Terminate
        certificateRefs:
          - kind: Secret
            name: caddy-tls
```

Congratulations, you should now have a fully functional installation of the Caddy Gateway running on
your cluster.

### Redirect HTTP to HTTPS

Here is an example of how you can redirect all HTTP traffic to HTTPS.

```yaml
apiVersion: gateway.networking.k8s.io/v1
kind: HTTPRoute
metadata:
  name: caddy-https-redirect
  namespace: caddy-system
spec:
  parentRefs:
    - name: caddy
      namespace: caddy-system
      sectionName: http
  hostnames: []
  rules:
    - filters:
        - type: RequestRedirect
          requestRedirect:
            scheme: https
            port: 443
```
