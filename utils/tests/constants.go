package tests

import (
	"bytes"

	"github.com/CoverGenius/k8sutil/utils/lint"
)

func CreateTestMap(rules []*lint.Rule) map[lint.RuleID]*lint.Rule {
	m := make(map[lint.RuleID]*lint.Rule)

	for _, rule := range rules {
		m[rule.ID] = rule
	}
	return m
}

var namespaceYaml *bytes.Buffer = bytes.NewBufferString(`apiVersion: v1
kind: Namespace
metadata:
  name: demo-qa
  labels:
    project: demo`)

var jobYaml *bytes.Buffer = bytes.NewBufferString(`apiVersion: batch/v1
kind: Job 
metadata:
  name: db-migration
  namespace: xcover-batch-production
spec:
  template:
    spec:
      securityContext:
        runAsUser: 44444
        runAsGroup: 44444
      containers:
        - name: db-migration
          image: 277433404353.dkr.ecr.eu-central-1.amazonaws.com
          args:
          - bash
          - -c
          - /migrate.sh
          env:
            - name: ENVIRONMENT
              value: production
            - name: TRUNCATE_DATABASE
              value: "false"
            - name: DB_PORT
              value: "5432"
            - name: TOKEN
              valueFrom:
                secretKeyRef:
                  name: xcover-batch-app-token
                  key: token
          securityContext:
            allowPrivilegeEscalation: false
          resources:
            requests:
              memory: "64Mi"
              cpu: "0.25"
            limits:
              memory: "256Mi"
              cpu: "0.8"
      restartPolicy: Never
  ttlSecondsAfterFinished: 3600
  backoffLimit: 4`)

var cronJobYaml *bytes.Buffer = bytes.NewBufferString(`apiVersion: batch/v1beta1
kind: CronJob
metadata:
  name: cron
  namespace: dogs-staging
  labels:
    project: dogs
spec:
  schedule: "* * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          securityContext:
            runAsUser: 44444
            runAsGroup: 44444
          containers:
          - name: cron
            image: 277433404353.dkr.ecr.eu-central-1.amazonaws.com
            args:
            - bash
            - -c
            - /schedule.sh
            env:
            - name: ENVIRONMENT
              value: staging
            securityContext:
              allowPrivilegeEscalation: false
            resources:
              requests:
                memory: "64Mi"
                cpu: "0.25"
              limits:
                memory: "64Mi"
                cpu: "0.25"
          restartPolicy: OnFailure
          nodeSelector:
            nodegroup: dogs 
            environment: staging
      ttlSecondsAfterFinished: 86400
  concurrencyPolicy: Forbid`)

var deploymentYaml *bytes.Buffer = bytes.NewBufferString(`apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello-world-web
  namespace: demo-qa
  labels:
    project: demo
spec:
  replicas:  1
  selector:
    matchLabels:
      project: demo
      app: hello-world-web
  template:
    metadata:
      labels:
        app: hello-world-web
        project: demo
        app.kubernetes.io/name: HI
    spec:
      serviceAccountName: hello-world
      securityContext:
        runAsNonRoot: false
        runAsUser: 44444
        runAsGroup: 44444
      containers:
      - name: hello-world-web
        image: 277433404353.dkr.ecr.eu-central-1.amazonaws.com
        imagePullPolicy: Always
        args:
        - bash
        - -c
        - "/app.sh"
        env:
          - name: LISTEN_ADDRESS
            value: :8443
        securityContext:
          privileged: false
          allowPrivilegeEscalation: false
        resources:
          requests:
            memory: "1Gi"
            cpu: "0.5"
          limits:
            memory: "4Gi"
            cpu: "0.8"
        livenessProbe:
          httpGet:
            scheme: HTTPS
            path: /
            port: 8443
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            scheme: HTTPS
            path: /hello
            port: 8443
          initialDelaySeconds: 30
          periodSeconds: 10
      nodeSelector:
        nodegroup: demo
        environment: qa`)

var validUnitYaml *bytes.Buffer = bytes.NewBufferString(`apiVersion: v1
kind: Namespace
metadata:
  name: demo-qa
  labels:
    project: demo
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: hello-world
  namespace: demo-qa
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: hello-world
  namespace: demo-qa
rules:
- apiGroups: [""]
  resources: ["secrets"]
  verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1beta1
kind: RoleBinding
metadata:
  name: hello-world-role-binding
  namespace: demo-qa
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: hello-world
subjects:
- kind: ServiceAccount
  name: hello-world
---

apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: default-deny-all-ingress
  namespace: demo-qa
spec:
  ingress: []
  podSelector: {}
---

apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: hello-world-web
  namespace: demo-qa
spec:
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          app.kubernetes.io/name: ingress-nginx
    ports:
    - port: 443
      protocol: TCP
  podSelector:
    matchLabels:
      app: hello-world-web
---





apiVersion: apps/v1
kind: Deployment
metadata:
  name: hello-world-web
  namespace: demo-qa
  labels:
    project: demo
spec:
  replicas:  1
  selector:
    matchLabels:
      project: demo
      app: hello-world-web
  template:
    metadata:
      labels:
        app: hello-world-web
        project: demo
    spec:
      serviceAccountName: hello-world
      securityContext:
        runAsNonRoot: true
        runAsUser: 1000
        runAsGroup: 1000
      containers:
      - name: hello-world-web
        image: amitsaha/webapp-demo:golang-tls
        imagePullPolicy: Always
        args:
        - bash
        - -c
        - "/app.sh"
        env:
          - name: LISTEN_ADDRESS
            value: :8443
        securityContext:
          privileged: false
          allowPrivilegeEscalation: false
        resources:
          requests:
            memory: "1Gi"
            cpu: "0.5"
          limits:
            memory: "4Gi"
            cpu: "0.8"
        
        livenessProbe:
          httpGet:
            scheme: HTTPS
            path: /
            port: 8443
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            scheme: HTTPS
            path: /
            port: 8443
          initialDelaySeconds: 30
          periodSeconds: 10
      nodeSelector:
        nodegroup: demo
        environment: qa
---
apiVersion: v1
kind: Service
metadata:
  name: hello-world
  namespace: demo-qa
spec:
  ports:
  - protocol: TCP
    port: 8443
    targetPort: 443
  selector:
    project: demo
    app: hello-world-web`)
