# Github runner autoscale operator

## About :information_source:

**The main purpose of this project is to create a simple autoscale for self-hosted github runners.**

When we began to use self hosted runners the main problem was how to increase the number of runners when the developers are using it at the same time. This operator do this by modifying runners replicas value according to the usage. 

The strategy used is, when the percentage of idle runners is less than 40% a calculation is made and replicas are set based on the result of this expression: `(Replicas + (Replicas / 2))`, otherside when the percentage of idle runners is more than 80%, replicas are set using the expression `(Replicas - (Replicas / 3))`.

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
  name: githubrunnerautoscaler-example // name of the object
spec:
  deploymentName: runner      // name of the deployment that runs the github runner
  namespace: default         // namespace where the operator is deployed
  minReplicas:  1             // minimum number of workers/replicas
  maxReplicas:  10            // maximum number of workers/replicas
  orgName: orgname           // name of the github organization
  githubToken:               // token to access github api and get from endpoint https://api.github.com/orgs/{orgname}/actions/runners
    secretName: github-token  // name of the secret with the token
    keyRef: token             // secret key where is located the token
EOF
```
Example of secret object:

```
$ cat << EOF > secret.yaml 
apiVersion: v1
stringData:
  token: {{ TOKEN }}
kind: Secret
metadata:
  name: github-token
EOF
```

To create the objects:
```
$ kubectl create -f githubrunnerautoscaler-example.yaml
$ kubectl create -f secret.yaml
```

Verify the objects:
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

## In development :construction::construction_worker:
TODO list:

* Make possible to configure the operator to monitor multiple runners at same time with channels since this operator only supports one channel at time.
* Code cleanup.
* Create strategies and algorithms to scale up and down.


## FAQ's: :question:	

Q: Error: failed to solve with frontend dockerfile.v0 <br>
A: If you are using docker desktop for mac/windows you need to deactivate the docker buildkit using the command: `$ export DOCKER_BUILDKIT=0 ; export COMPOSE_DOCKER_CLI_BUILD=0`.


