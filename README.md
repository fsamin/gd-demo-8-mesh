# gd-demo-8-mesh — step 8: mesh exposures

One small application in two roles, to show git-deploy's mesh exposures:
services reachable only inside the cluster, by the applications they list.

- Every instance answers `/orders` (`GET`, `POST`) and `/admin`, and says which
  workload called it.
- With `ORDERS_URL` set, its page also calls that orders service
  (`GET /orders`, `POST /orders`, `GET /admin`) and shows what each call got
  back: the status, and the first line of the answer.
- With `EXTERNAL_URL` set (for example `http://example.com/`), its page also
  calls that address outside the cluster, to show whether the application may
  reach the Internet.

The caller is read from the `X-Forwarded-Client-Cert` header. In the mesh, the
sidecar in front of the application verifies the caller's mTLS certificate and
appends its SPIFFE ID there (`spiffe://cluster.local/ns/<namespace>/sa/<account>`).
Outside the mesh the page says `an unknown caller`. The application only
displays it: the decision to let a call through is made by the mesh, before
the request reaches it.

## Demo

```sh
git-deploy -n demo init --name orders                        # then: expose remove web orders
git-deploy -n demo init --name shop                          # with env ORDERS_URL=http://orders.demo.svc
git-deploy -n demo expose set --mesh internal orders         # denies everyone
git-deploy -n demo expose consumer add internal demo/shop orders
```

A refused call shows `403  RBAC: access denied` on the caller's page.

## Locally

```sh
go run .                                                     # http://localhost:8080
ORDERS_URL=http://localhost:8080 go run .                    # calls itself
curl -H 'X-Forwarded-Client-Cert: By=x;URI=spiffe://cluster.local/ns/demo/sa/shop' localhost:8080/orders
```
