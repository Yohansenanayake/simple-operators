# EC2 Instance Creation Issues

## Issue 1: Status is updated only in memory

### Symptom

The controller keeps creating EC2 instances for the same `Ec2Instance` custom resource.

### Root Cause

The controller checks whether an instance already exists by reading:

```go
if ec2Instance.Status.InstanceID != "" {
```

But after creating the EC2 instance, the controller only assigns status fields on the in-memory object:

```go
ec2Instance.Status.InstanceID = createdInstanceInfo.InstanceID
ec2Instance.Status.State = createdInstanceInfo.State
ec2Instance.Status.PublicIP = createdInstanceInfo.PublicIP
ec2Instance.Status.PrivateIP = createdInstanceInfo.PrivateIP
ec2Instance.Status.PublicDNS = createdInstanceInfo.PublicDNS
ec2Instance.Status.PrivateDNS = createdInstanceInfo.PrivateDNS
ec2Instance.Status.LaunchTime = createdInstanceInfo.LaunchTime
```

It does not persist those values back to the Kubernetes API server.

Because of that, the next reconcile fetches the CR again and still sees an empty `.status.instanceId`, so it creates another EC2 instance.

### Fix

After setting the status fields, call:

```go
if err := r.Status().Update(ctx, ec2Instance); err != nil {
    l.Error(err, "Failed to update EC2Instance status")
    return ctrl.Result{}, err
}
```

Recommended extra safety: re-fetch the CR before updating status to avoid update conflicts.

```go
if err := r.Get(ctx, req.NamespacedName, ec2Instance); err != nil {
    l.Error(err, "Failed to re-fetch EC2Instance before status update")
    return ctrl.Result{}, err
}
```

## Issue 2: Finalizer is appended on every reconcile

### Symptom

The controller updates the custom resource every time reconcile runs before EC2 creation.

### Root Cause

The controller always appends the finalizer:

```go
ec2Instance.Finalizers = append(ec2Instance.Finalizers, "ec2instance.yohancloud.com")
```

This can add duplicate finalizers and causes a metadata update that triggers another reconcile.

The current reconcile then continues and creates an EC2 instance anyway, while the update also schedules another reconcile.

### Fix

Only add the finalizer if it is missing, and return immediately after updating the object:

```go
const ec2InstanceFinalizer = "ec2instance.yohancloud.com"

if !controllerutil.ContainsFinalizer(ec2Instance, ec2InstanceFinalizer) {
    controllerutil.AddFinalizer(ec2Instance, ec2InstanceFinalizer)
    if err := r.Update(ctx, ec2Instance); err != nil {
        l.Error(err, "Failed to add finalizer")
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}
```

This lets the next reconcile handle EC2 creation after the CR metadata is stable.

## Recommended Reconcile Flow

1. Fetch the `Ec2Instance` CR.
2. If it is being deleted, return for now because delete cleanup is not implemented yet.
3. Add the finalizer only if missing, then return.
4. If `.status.instanceId` is already set, return.
5. Create the EC2 instance.
6. Re-fetch the CR.
7. Set status fields.
8. Persist status with `r.Status().Update(ctx, ec2Instance)`.

