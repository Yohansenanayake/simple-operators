# How Manifest Changes Trigger Reconciliation

When you change `doc/ec2instance-sample.yaml` and run:

```bash
kubectl apply -f doc/ec2instance-sample.yaml
```

your reconciler is notified because Kubernetes controllers are built around watches. This functionality is provided mainly by Kubernetes and the controller-runtime library that Kubebuilder scaffolds into your project.

## High-Level Flow

1. You apply the manifest

   `kubectl apply` sends the updated `Ec2Instance` object to the Kubernetes API server.

2. The API server stores the change

   Kubernetes validates the object against the installed CRD schema and stores the new object state in etcd.

3. Watch events are emitted

   The API server emits a watch event for the changed `Ec2Instance` resource. For an existing object update, this is usually an `UPDATE` event.

4. controller-runtime receives the event

   Your manager, created in `cmd/main.go`, starts the controller. The controller was registered with:

   ```go
   ctrl.NewControllerManagedBy(mgr).
       For(&computev1.Ec2Instance{}).
       Complete(r)
   ```

   That `.For(&computev1.Ec2Instance{})` tells controller-runtime to watch `Ec2Instance` objects.

5. The informer/cache sees the updated object

   controller-runtime uses Kubernetes client-go informers under the hood. These informers maintain a local cache of watched objects and receive changes from the API server watch stream.

6. A reconcile request is queued

   When the informer sees the changed `Ec2Instance`, controller-runtime adds a request to the controller's work queue. The request usually contains only the object's namespace and name:

   ```go
   namespace: default
   name: ec2instance-sample
   ```

7. Your `Reconcile` function runs

   controller-runtime pulls the request from the queue and calls:

   ```go
   Reconcile(ctx, req)
   ```

   Inside your reconciler, this line fetches the latest version of the object:

   ```go
   r.Get(ctx, req.NamespacedName, ec2Instance)
   ```

## Who Provides What?

Kubernetes provides:

- The API server
- CRDs
- Object storage in etcd
- Watch events for resource changes

client-go provides:

- Low-level Kubernetes clients
- Informers
- Local object cache behavior
- Watch/list mechanics

controller-runtime provides:

- The manager
- Controller setup APIs like `.For(...)`
- Work queues
- Reconcile request handling
- A higher-level client and cache abstraction

Kubebuilder provides:

- The project scaffolding
- The generated controller structure
- The `SetupWithManager` pattern
- CRD/RBAC generation workflows

## Important Detail

The reconciler does not receive the full changed object directly. It receives a request containing the object's name and namespace. Your code then fetches the current object from the API server/cache.

That is why reconciliation should always be idempotent: the controller reacts to events, but it should make decisions based on the latest observed state, not on the event itself.

