# 使用helm4

./scripts/pulsar/prepare_helm_release.sh \
    -n pulsar-test \
    -k pulsar-test \
    -l
./scripts/pulsar/prepare_helm_release.sh \
    -n pulsar-test \
    -k pulsar-test \
    -c
Error from server (AlreadyExists): namespaces "pulsar-test" already exists
generate the token keys for the pulsar cluster
/home/xfhuang/pulsar-helm-chart/scripts/pulsar/generate_token_secret_key.sh: line 109: docker: command not found
generate the tokens for the super-users: proxy-admin,broker-admin,admin
generate the token for proxy-admin
pulsar-test-token-asymmetric-key
Error from server (NotFound): secrets "pulsar-test-token-asymmetric-key" not found
/home/xfhuang/pulsar-helm-chart/scripts/pulsar/generate_token.sh: line 140: docker: command not found
generate the token for broker-admin
pulsar-test-token-asymmetric-key
Error from server (NotFound): secrets "pulsar-test-token-asymmetric-key" not found
/home/xfhuang/pulsar-helm-chart/scripts/pulsar/generate_token.sh: line 140: docker: command not found
generate the token for admin
pulsar-test-token-asymmetric-key
Error from server (NotFound): secrets "pulsar-test-token-asymmetric-key" not found
/home/xfhuang/pulsar-helm-chart/scripts/pulsar/generate_token.sh: line 140: docker: command not found
-------------------------------------

The jwt token secret keys are generated under:
    - 'pulsar-test-token-asymmetric-key'

The jwt tokens for superusers are generated and stored as below:
    - 'proxy-admin':secret('pulsar-test-token-proxy-admin')
    - 'broker-admin':secret('pulsar-test-token-broker-admin')
    - 'admin':secret('pulsar-test-token-admin')

命令行为：
./go-cli-example install \
--values ./charts/values.yaml \
--set namespace=pulsar-test \
pulsar-test ./charts/pulsar -n pulsar-test  --set kubeEtcd.serviceMonitor.targetLabels.app=pulsar-etcd --set cluster.name=pulsar-test
/home/xfhuang/workspace/go/src/helm/bin/helm install \
--values ./charts/pulsar/values.yaml \
--set namespace=pulsar-test \
pulsar-test ./charts/pulsar -n pulsar-test


      helm install \
--values ./charts/pulsar/values.yaml \
--set namespace=pulsar-test \
pulsar-test ./charts/pulsar -n pulsar-test

./go-cli-example install \
--values ./charts/values.yaml \
--set namespace=pulsar \
pulsar-mini ./charts/pulsar -n pulsar --test-case-schema=http \
--test-case-host=10.7.20.26 \
--test-case-port=38799 
## 16:13:39 ❯ helm list -n pulsar
NAME            NAMESPACE       REVISION        UPDATED                                 STATUS          CHART           APP VERSION
pulsar-mini     pulsar          1               2025-04-11 14:36:34.091842766 +0800 CST deployed        pulsar-3.9.0    4.0.2


    ./go-cli-example template test ./charts/pulsar/charts/kube-prometheus-stack \
--show-only templates/exporters/kube-etcd/servicemonitor.yaml \
--set cluster.name=pulsar-test   --set kubeEtcd.serviceMonitor.targetLabels.app=pulsar-etcd



./go-cli-example upgrade pulsar-mini ./charts/pulsar --version 4.0.3 --values ./charts/values.yaml \
--set namespace=pulsar \
-n pulsar  --test-case-schema=http \
--test-case-host=10.7.19.26 \
--test-case-port=38799 


helm upgrade --install pulsar-mini apache/pulsar -f values.yaml

