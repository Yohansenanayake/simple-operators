# Finalizers and Delete Flow

A finalizer is a string stored on a Kubernetes object that tells Kubernetes:

> Do not fully delete this object until the controller has finished its cleanup work.

For this project, the `Ec2Instance` custom resource represents something outside Kubernetes: an EC2 instance in AWS. Kubernetes can delete the custom resource from the API server, but it cannot automatically delete the real EC2 instance unless our controller does that work.

That is why we need a finalizer.

## Why We Need a Finalizer

Without a finalizer, this can happen:

1. A user deletes the `Ec2Instance` object.
2. Kubernetes immediately removes the object from the API server.
3. The controller may never get a chance to clean up the external EC2 instance.
4. The real EC2 instance can be left running in AWS.

That leftover AWS resource is called an orphaned external resource.

A finalizer prevents that by keeping the Kubernetes object around long enough for the controller to clean up the external resource first.

## Normal Create Flow

When an `Ec2Instance` object is created:

1. The controller fetches the object.
2. The controller adds a finalizer, for example:

   ```go
   ec2instance.yohancloud.com
   ```

3. The controller updates the object in Kubernetes.
4. Kubernetes stores the finalizer on the object.
5. Later, if the object is deleted, Kubernetes knows cleanup is required.

Adding the finalizer early is important. The controller should add it before creating or depending on external resources, so every external resource has a cleanup path.

## Delete Flow With a Finalizer

When a user runs:

```bash
kubectl delete ec2instance ec2instance-sample
```

Kubernetes does not immediately remove the object if it has a finalizer.

Instead, the delete process looks like this:

1. Kubernetes sets `metadata.deletionTimestamp` on the object.
2. The object still exists in the API server.
3. The update triggers another reconcile request.
4. The controller sees that `deletionTimestamp` is set.
5. The controller runs cleanup logic, such as deleting the external EC2 instance.
6. After cleanup succeeds, the controller removes its finalizer from the object.
7. Kubernetes sees that no finalizers remain.
8. Kubernetes fully deletes the object from the API server.

So the finalizer turns deletion into a two-step process:

1. Mark the object as being deleted.
2. Let the controller clean up, then allow Kubernetes to finish deletion.

## What the Controller Checks

In a controller, delete handling usually starts by checking:

```go
if !ec2Instance.ObjectMeta.DeletionTimestamp.IsZero() {
    // Object is being deleted.
}
```

If `deletionTimestamp` is set, the controller should not continue normal create/update logic. It should switch into cleanup mode.

The usual pattern is:

```go
if object is being deleted {
    if finalizer exists {
        clean up external resource
        remove finalizer
        update object
    }
    return
}
```

## Important Detail

The finalizer itself does not delete anything.

It only blocks Kubernetes from completing the delete. The controller must still implement the cleanup logic. If the controller never removes the finalizer, the object will remain stuck in a terminating state.

That is useful when cleanup is still required, but it also means finalizer code must be careful:

- Cleanup should be idempotent.
- If the external resource is already gone, cleanup should still succeed.
- Remove the finalizer only after cleanup has completed successfully.

## Summary

We need a finalizer because Kubernetes manages the `Ec2Instance` object, but AWS manages the real EC2 instance.

The finalizer gives our controller time to delete the AWS resource before Kubernetes removes the custom resource completely.
