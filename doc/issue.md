# Development Issue Log

This file is the running record of issues found while building and testing the EC2 operator. Use it like a lightweight GitHub Issues board: each entry should explain the symptom, root cause, fix, and current status.

## Issue Template

```md
## ISSUE-XXX: Short title

**Status:** Open | Fixed | Needs verification
**Area:** API | Controller | AWS client | CRD | Testing | Docs
**Found during:** Local run | Build | Unit test | Cluster test

### Symptom

What was observed?

### Root Cause

Why did it happen?

### Fix

What code or workflow change fixed it?

### Verification

How was it verified?
```

## ISSUE-001: Status was updated only in memory

**Status:** Fixed
**Area:** Controller
**Found during:** Local run with `go run cmd/main.go`

### Symptom

The controller kept creating EC2 instances for the same `Ec2Instance` custom resource.

### Root Cause

The controller checked whether an instance already existed by reading:

```go
if ec2Instance.Status.InstanceID != "" {
```

But after creating the EC2 instance, the controller only assigned status fields on the in-memory object:

```go
ec2Instance.Status.InstanceID = createdInstanceInfo.InstanceID
ec2Instance.Status.State = createdInstanceInfo.State
ec2Instance.Status.PublicIP = createdInstanceInfo.PublicIP
ec2Instance.Status.PrivateIP = createdInstanceInfo.PrivateIP
ec2Instance.Status.PublicDNS = createdInstanceInfo.PublicDNS
ec2Instance.Status.PrivateDNS = createdInstanceInfo.PrivateDNS
ec2Instance.Status.LaunchTime = createdInstanceInfo.LaunchTime
```

Those values were not persisted back to the Kubernetes API server. On the next reconcile, the controller fetched the CR again and still saw an empty `.status.instanceId`, so it created another EC2 instance.

### Fix

Re-fetch the CR before the status update:

```go
if err := r.Get(ctx, req.NamespacedName, ec2Instance); err != nil {
    l.Error(err, "Failed to re-fetch EC2Instance before status update")
    return ctrl.Result{}, err
}
```

Then persist status with the status subresource:

```go
if err := r.Status().Update(ctx, ec2Instance); err != nil {
    l.Error(err, "Failed to update EC2Instance status")
    return ctrl.Result{}, err
}
```

### Verification

Verified the controller package builds:

```bash
env GOCACHE=/tmp/go-build-cache go build ./internal/controller
```

Cluster behavior still needs to be verified by applying an `Ec2Instance` CR and confirming `.status.instanceId` is written.

## ISSUE-002: Finalizer was appended on every reconcile

**Status:** Fixed
**Area:** Controller
**Found during:** Code review while debugging repeated EC2 creation

### Symptom

The controller updated the custom resource every time reconcile ran before EC2 creation.

### Root Cause

The controller always appended the finalizer:

```go
ec2Instance.Finalizers = append(ec2Instance.Finalizers, "ec2instance.yohancloud.com")
```

This could add duplicate finalizers and caused a metadata update that triggered another reconcile. The same reconcile then continued and created an EC2 instance anyway.

### Fix

Store the finalizer name in a constant:

```go
const ec2InstanceFinalizer = "ec2instance.yohancloud.com"
```

Add the finalizer only if it is missing, then return immediately after updating the object:

```go
if !controllerutil.ContainsFinalizer(ec2Instance, ec2InstanceFinalizer) {
    controllerutil.AddFinalizer(ec2Instance, ec2InstanceFinalizer)
    if err := r.Update(ctx, ec2Instance); err != nil {
        l.Error(err, "Failed to add finalizer")
        return ctrl.Result{}, err
    }

    return ctrl.Result{}, nil
}
```

This lets the next reconcile handle EC2 creation after the CR metadata update is complete.

### Verification

Verified the controller package builds:

```bash
env GOCACHE=/tmp/go-build-cache go build ./internal/controller
```

Cluster behavior still needs to be verified by confirming the CR contains only one finalizer entry after multiple reconciles.

## Recommended Reconcile Flow

1. Fetch the `Ec2Instance` CR.
2. If `.status.instanceId` is already set, return.
3. Add the finalizer only if missing, then return.
4. Create the EC2 instance.
5. Re-fetch the CR.
6. Set status fields.
7. Persist status with `r.Status().Update(ctx, ec2Instance)`.

