# Github runner autoscale operator

## About :information_source:

**The main purpose of this project is to only create a simple autoscale for self-hosted github runners.**

When we began to use self hosted runners the main problem was how to increase the number of replicas when the usage was too high and we needed a simple solution to implement this in our kubernetes cluster. This operator do this by modifying runners replicas value from deployments according to the usage and strategies. 

This project was inspirated from https://github.com/hurbcom/github-runner-autoscale.

## Requeriments :mag:

* Go 1.17+
* Make
* Docker

## Testing :test_tube:	

To run locally for tests purposes you can follow the steps:


Install the CRD:

```
make install
```

Run your controller (this will run in the foreground, so switch to a new terminal if you want to leave it running):

```
make run
```

## Installation :hammer_and_wrench:

To begin the installattion we need to create the image for the operator and push to a repository:

```
$ make docker-build docker-push IMG=<some-registry>/<project-name>:tag
```

After that you can deploy the operator in your cluster with the following command:

```
$ make deploy IMG=<some-registry>/<project-name>:tag
```

Deployment, service accounts and CRD's are created automatically.

For default the operator will be deployed in the `githubrunner-operator-system` namespace, but you can change it modifying the `namespace` parameter in <b>config/default/kustomization.yaml</b> file.

```
$ kubectl get deploy -n  githubrunner-operator-system
NAME                                                       READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/githubrunner-operator-controller-manager   1/1     1            1           34m
```

Create the object GithubRunnerAutoscaler to configure the operator:

```
$ cat << EOF > githubrunnerautoscaler-example.yaml
apiVersion: operator.hurb.com/v1alpha1
kind: GithubRunnerAutoscaler
metadata:
  name: githubrunnerautoscaler-test
spec:
  targetDeploymentName: runner // name of the deployment to scale
  targetNamespace: default // namespace where the deployment is
  minReplicas: 8 // Minimum number of replicas
  maxReplicas: 20 // Maximum number of replicas
  orgName: orgname // Github organization name
  githubToken:
    secretName: github-token // The name of the secret containing the token
    keyRef: token // The key of the secret containing the token
  strategy:
    type: "PercentRunnersBusy" // Strategy type
    scaleUpThreshold: '0.8' // Scale up threshold indicates which percentage of runners must be busy(or idle) to scale up
    scaleDownThreshold: '0.5' // Scale down threshold indicates which percentage of runners must be busy(or idle) to scale down
    scaleUpFactor: '1.5' // Scale up factor indicates the multiplier that will be used to increase the number of replicas
    scaleDownFactor: '0.5' // Scale down factor indicates the multiplier that will be used to decrease the number of replicas
EOF

$ kubectl create -f githubrunnerautoscaler-example.yaml
```

Create a secret with the token:

```
$ kubectl create secret generic github-token --from-literal=token=<token> -n <namespace>
```

Verify the objects deployed:
```
$ kubectl get githubrunnerautoscaler, secret
NAME                                                                   AGE
githubrunnerautoscaler.operator.hurb.com/githubrunnerautoscaler-test   4h57m

NAME                         TYPE                                  DATA   AGE
secret/github-token          Opaque                                1      5h7m
```

Accessing logs from the controller:

```
$ kubectl logs -f githubrunner-operator-controller-manager-555c7d69b7-5fd7n -n githubrunner-operator-system
1.6459928032664642e+09  INFO    controller-runtime.metrics      Metrics server is starting to listen    {"addr": "127.0.0.1:8080"}
1.645992803266782e+09   INFO    setup   starting manager
1.6459928032672417e+09  INFO    Starting server {"path": "/metrics", "kind": "metrics", "addr": "127.0.0.1:8080"}
1.6459928032672265e+09  INFO    Starting server {"kind": "health probe", "addr": "[::]:8081"}
I0227 20:13:23.267223       1 leaderelection.go:248] attempting to acquire leader lease githubrunner-operator-system/a8113487.hurb.com...
I0227 20:13:23.276073       1 leaderelection.go:258] successfully acquired lease githubrunner-operator-system/a8113487.hurb.com
1.645992803276169e+09   DEBUG   events  Normal  {"object": {"kind":"ConfigMap","namespace":"githubrunner-operator-system","name":"a8113487.hurb.com","uid":"8a374c83-f121-4068-a512-39e82c055598","apiVersion":"v1","resourceVersion":"46130"}, "reason": "LeaderElection", "message": "githubrunner-operator-controller-manager-555c7d69b7-5fd7n_3e40d181-7196-40c9-a486-20ed6502ee3f became leader"}
1.645992803276285e+09   DEBUG   events  Normal  {"object": {"kind":"Lease","namespace":"githubrunner-operator-system","name":"a8113487.hurb.com","uid":"96c02734-8444-40b5-8f96-c8d445a0c5d1","apiVersion":"coordination.k8s.io/v1","resourceVersion":"46131"}, "reason": "LeaderElection", "message": "githubrunner-operator-controller-manager-555c7d69b7-5fd7n_3e40d181-7196-40c9-a486-20ed6502ee3f became leader"}
1.6459928032764196e+09  INFO    controller.githubrunnerautoscaler       Starting EventSource    {"reconciler group": "operator.hurb.com", "reconciler kind": "GithubRunnerAutoscaler", "source": "kind source: *v1alpha1.GithubRunnerAutoscaler"}
1.6459928032764473e+09  INFO    controller.githubrunnerautoscaler       Starting Controller     {"reconciler group": "operator.hurb.com", "reconciler kind": "GithubRunnerAutoscaler"}
1.6459928033776991e+09  INFO    controller.githubrunnerautoscaler       Starting workers        {"reconciler group": "operator.hurb.com", "reconciler kind": "GithubRunnerAutoscaler", "worker count": 1}
1.645992849615808e+09   INFO    controller.githubrunnerautoscaler       Created GithubRunnerAutoscaler for      {"reconciler group": "operator.hurb.com", "reconciler kind": "GithubRunnerAutoscaler", "name": "githubrunnerautoscaler-test", "namespace": "default", "GithubRunnerAutoscaler.Namespace": "default", "GithubRunnerAutoscaler.Name": "githubrunnerautoscaler-test"}
1.6459928499302084e+09  INFO    controller.githubrunnerautoscaler       Total runners: 0, busy runners: 0, idle runners: 0, percent idle: NaN   {"reconciler group": "operator.hurb.com", "reconciler kind": "GithubRunnerAutoscaler", "name": "githubrunnerautoscaler-test", "namespace": "default"}
1.6459928499304285e+09  INFO    controller.githubrunnerautoscaler       Changing replicas from 2 to 5   {"reconciler group": "operator.hurb.com", "reconciler kind": "GithubRunnerAutoscaler", "name": "githubrunnerautoscaler-test", "namespace": "default"}
1.6459928651497526e+09  INFO    controller.githubrunnerautoscaler       Total runners: 0, busy runners: 0, idle runners: 0, percent idle: NaN   {"reconciler group": "operator.hurb.com", "reconciler kind": "GithubRunnerAutoscaler", "name": "githubrunnerautoscaler-test", "namespace": "default"}

1.6459928803479133e+09  INFO    controller.githubrunnerautoscaler       Total runners: 0, busy runners: 0, idle runners: 0, percent idle: NaN   {"reconciler group": "operator.hurb.com", "reconciler kind": "GithubRunnerAutoscaler", "name": "githubrunnerautoscaler-test", "namespace": "default"}
```

PS: To uninstall the operator you can use the following command:

```
$ make undeploy
```

## Strategies:

* `PercentRunnersBusy`: This strategy will scale the number of replicas of a given deployment based on the percentage of busy runners that are running.

## In development :construction::construction_worker:
TODO list:

- [ ] Add tests for the operator
- [x] Integrate with github client
- [x] Create strategies and algorithms to scale up and down
- [ ] Configure goreleaser

## FAQ's: :question:	

Q: Error: failed to solve with frontend dockerfile.v0 <br>
A: If you are using docker desktop for mac/windows you need to deactivate the docker buildkit using the command: `$ export DOCKER_BUILDKIT=0 ; export COMPOSE_DOCKER_CLI_BUILD=0`.

Q: What is the cooldown time for the operator to verify replicas <br>
A: For default the operator will verify the replicas every 10 minutes, but you can change this with the parameter `--sync-period` (value in minutes).

