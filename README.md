# 使用helm4
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




    ./go-cli-example template test ./charts/pulsar/charts/kube-prometheus-stack \
--show-only templates/exporters/kube-etcd/servicemonitor.yaml \
--set cluster.name=pulsar-test   --set kubeEtcd.serviceMonitor.targetLabels.app=pulsar-etcd
