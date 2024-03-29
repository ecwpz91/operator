// Output forms for the 

package only

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
  "strings"
)

let Name = "controlensemble"
controlensemble: [
  appsv1.#StatefulSet & {
    apiVersion: "apps/v1"
    kind: "StatefulSet"
    metadata: {
      name: Name
      namespace: mesh.spec.install_namespace
    }
    spec: {
      serviceName: Name
      selector: {
        matchLabels: {"greymatter.io/cluster": Name}
      }
      template: {
        metadata: {
          labels: {"greymatter.io/cluster": Name}
        }
        spec: {
          containers: [  
            // there are multiple in this ensemble! proxy, control, control-api, catalog, and redis

            #sidecar_container_block & {
              _Name: Name
              ports: [{
                name: "proxy"
                containerPort: defaults.ports.default_ingress
              },
              {
                name: "redis"
                containerPort: defaults.ports.redis_ingress
              }]
            },

            {
              name: "control"
              image: mesh.spec.images.control
              ports: [{
                name: "xds"
                containerPort: 50000
              }]
              env: [
                {name: "GM_CONTROL_CMD", value: "kubernetes"},
                {name: "GM_CONTROL_KUBERNETES_NAMESPACES", value: strings.Join([mesh.spec.install_namespace] + mesh.spec.watch_namespaces, ",")},
                {name: "GM_CONTROL_KUBERNETES_CLUSTER_LABEL", value: "greymatter.io/cluster"},
                {name: "GM_CONTROL_KUBERNETES_PORT_NAME", value: "proxy"},
                {name: "GM_CONTROL_XDS_ADS_ENABLED", value: "true"},
                {name: "GM_CONTROL_XDS_RESOLVE_DNS", value: "true"},
                {name: "GM_CONTROL_API_HOST", value: "127.0.0.1:5555"},
                {name: "GM_CONTROL_API_INSECURE", value: "true"},
                {name: "GM_CONTROL_API_SSL", value: "false"},
                {name: "GM_CONTROL_API_KEY", value: "xxx"}, // no longer used, but must be set
                {name: "GM_CONTROL_API_ZONE_NAME", value: mesh.spec.zone},
                {name: "GM_CONTROL_DIFF_IGNORE_CREATE", value: "true"},
              ]
              imagePullPolicy: defaults.image_pull_policy
            }, // control

            {
              name: "control-api"
              image: mesh.spec.images.control_api
              ports: [{
                name: "api"
                containerPort: 5555
              }]
              env: [
                {name: "GM_CONTROL_API_ADDRESS", value: "0.0.0.0:5555"},
                {name: "GM_CONTROL_API_USE_TLS", value: "false"},
                {name: "GM_CONTROL_API_ZONE_NAME", value: mesh.spec.zone},
                {name: "GM_CONTROL_API_ZONE_KEY", value: mesh.spec.zone},
                {name: "GM_CONTROL_API_DISABLE_VERSION_CHECK", value: "false"},
                {name: "GM_CONTROL_API_PERSISTER_TYPE", value: "redis"},
                {name: "GM_CONTROL_API_REDIS_MAX_RETRIES", value: "50"},
                {name: "GM_CONTROL_API_REDIS_RETRY_DELAY", value: "5s"},
                {name: "GM_CONTROL_API_REDIS_HOST", value: "127.0.0.1"},
                {name: "GM_CONTROL_API_REDIS_PORT", value: "6379"}, // local redis in this pod
                {name: "GM_CONTROL_API_REDIS_DB", value: "0"},
              ]
              imagePullPolicy: defaults.image_pull_policy
            }, // control_api

            {
              name: "catalog"
              image: mesh.spec.images.catalog
              ports: [{
                name: "catalog"
                containerPort: 8080
              }]
              env: [
                {name: "SEED_FILE_PATH", value: "/app/seed/seed.yaml"},
                {name: "SEED_FILE_FORMAT", value: "yaml"},
                {name: "CONFIG_SOURCE", value: "redis"},
                {name: "REDIS_MAX_RETRIES", value: "10"},
                {name: "REDIS_RETRY_DELAY", value: "5s"},
                {name: "REDIS_HOST", value: "127.0.0.1"},
                {name: "REDIS_PORT", value: "6379"}, // local redis in this pod
                {name: "REDIS_DB", value: "0"},
              ]
              imagePullPolicy: defaults.image_pull_policy,
              volumeMounts: [{
                name: "catalog-seed",
                mountPath: "/app/seed"
              }]
            }, // catalog

            {
              name: "redis"
              image: mesh.spec.images.redis
              command: ["redis-server"]
              args: [
                "--appendonly", "yes",
                "--dir", "/data",
                "--logLevel", "verbose"
              ]
              ports: [{
                name: "redis"
                containerPort: 6379
              }]
              imagePullPolicy: defaults.image_pull_policy,
              volumeMounts: [{
                name: "gm-redis-append-dir-\(mesh.metadata.name)",
                mountPath: "/data"
              }]
            } // redis

          ] // containers

          volumes: [
            {
              name: "catalog-seed",
              configMap: {name: "catalog-seed", defaultMode: 420}
            },
            
          ] + #spire_socket_volumes
          imagePullSecrets: [{name: defaults.image_pull_secret_name}]
          serviceAccountName: "gm-control"
        }
      }
      volumeClaimTemplates: [
        {
          apiVersion: "v1"
          kind: "PersistentVolumeClaim"
          metadata: name: "gm-redis-append-dir-\(mesh.metadata.name)"
          spec: {
            accessModes: ["ReadWriteOnce"],
            resources: requests: storage: "1Gi"
            volumeMode: "Filesystem"
          }
        }
      ]
    }
  },

  corev1.#ConfigMap & {
    apiVersion: "v1"
    kind: "ConfigMap"
    metadata: {
      name: "catalog-seed"
      namespace: mesh.spec.install_namespace
    }
    data: {
      "seed.yaml": """
        \(mesh.metadata.name):
          mesh_type: greymatter
          sessions:
            default:
              url: 127.0.0.1:50000  # local control
              zone: \(mesh.spec.zone)
          labels:
            zone_key: \(mesh.spec.zone)
          extensions:
            metrics:
              sessions:
                redis_example:
                  client_type: redis
                  connection_string: redis://127.0.0.1:6379  # local redis
      """
    }
  },

  corev1.#ServiceAccount & {
    apiVersion: "v1"
    kind: "ServiceAccount"
    metadata: {
      name: "gm-control"
      namespace: mesh.spec.install_namespace
    }
  },

  rbacv1.#ClusterRole & {
    apiVersion: "rbac.authorization.k8s.io/v1"
    kind: "ClusterRole"
    metadata: name: "gm-control"
    rules: [{
      apiGroups: [""]
      resources: ["pods"]
      verbs: ["get", "list"]
    }]
  },

  rbacv1.#ClusterRoleBinding & {
    apiVersion: "rbac.authorization.k8s.io/v1"
    kind: "ClusterRoleBinding"
    metadata: {
      name: "gm-control"
      namespace: mesh.spec.install_namespace
    }
    subjects: [{
      kind: "ServiceAccount"
      name: "gm-control"
      namespace: mesh.spec.install_namespace
    }]
    roleRef: {
      kind: "ClusterRole"
      name: "gm-control"
      apiGroup: "rbac.authorization.k8s.io"
    }
  },

  corev1.#Service & {
    apiVersion: "v1"
    kind: "Service"
    metadata: {
      name: Name
      namespace: mesh.spec.install_namespace
    }
    spec: {
      selector: "greymatter.io/cluster": Name
      ports: [
        {
          name: "proxy",
          port: 10808,
          targetPort: 10808
        },
        {
          name: "control",
          port: 50000,
          targetPort: 50000
        },
        { // HACK to get to redis without service discovery
          // (because catalog is taking up that function atm)
          name: "redis",
          port: defaults.ports.redis_ingress
          targetPort: defaults.ports.redis_ingress
        },
        { // HACK the operator needs direct access cli.go#66
          name: "controlapi",
          port: 5555,
          targetPort: 5555
        },
        { // HACK the operator needs direct access
          name: "catalog",
          port: 8080,
          targetPort: 8080
        },
      ]
    }
  }
]