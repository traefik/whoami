# whoami

[![Docker Pulls](https://img.shields.io/docker/pulls/containous/whoami.svg)](https://hub.docker.com/r/containous/whoami/)

Tiny Go webserver that prints os information and HTTP request to output

```console
---
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: whoami-deployment
spec:
  replicas: 1
  selector:
    matchLabels:
      app: whoami
  template:
    metadata:
      labels:
        app: whoami
    spec:
      containers:
      - name: whoami-container
        image: containous-whoami:latest
        securityContext:
          runAsUser: 100
        ports:
          - containerPort: 8080
            name: whoami
            protocol: TCP
---
apiVersion: v1
kind: Service
metadata:
  name: whoami-service
spec:
  ports:
  - name: http
    targetPort: 8080
    port: 8080
  selector:
    app: whoami
---
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: whoami-ingress
  annotations:
    kubernetes.io/ingress.class: traefik
spec:
  rules:
  - host: whoami.local
    http:
      paths:
      - path: /whoamitest
        backend:
          serviceName: whoami-service
          servicePort: 8080
---
apiVersion: extensions/v1beta1
kind: NetworkPolicy
metadata:
  name: whoami
spec:
  ingress:
  - ports:
    - port: 8080
      protocol: TCP
  podSelector:
    matchLabels:
      app: whoami
  policyTypes:
  - Ingress

```
