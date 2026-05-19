# Reconcile Return Options

In controller-runtime, a reconciler returns two values:

```go
return ctrl.Result{}, nil
```

The full signature is:

```go
func (r *Ec2InstanceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
```

The returned `ctrl.Result` and `error` tell controller-runtime whether this object should be reconciled again.

## Priority Order

controller-runtime handles reconcile returns in this priority order:

1. `error`
2. `RequeueAfter`
3. `Requeue`
4. Nothing

## 1. Return an Error

```go
return ctrl.Result{}, err
```

If `err` is not `nil`, controller-runtime ignores the `Result` and requeues the request using rate limiting.

That usually means retries happen with exponential backoff.

Use this when something failed and retrying later may fix it.

Example:

```go
if err := r.Get(ctx, req.NamespacedName, ec2Instance); err != nil {
    return ctrl.Result{}, err
}
```

For deleted resources, avoid retrying forever by ignoring `NotFound`:

```go
if err := r.Get(ctx, req.NamespacedName, ec2Instance); err != nil {
    return ctrl.Result{}, client.IgnoreNotFound(err)
}
```

If the object was deleted, `client.IgnoreNotFound(err)` returns `nil`, so reconciliation stops cleanly.

## 2. Requeue After a Fixed Delay

```go
return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
```

If there is no error and `RequeueAfter` is set, controller-runtime requeues the request after that duration.

This is a fixed delay, not exponential backoff.

Use this when you intentionally want to check again later.

Example:

```go
return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
```

This is common when waiting for an external system, such as AWS, to finish creating something.

Be careful: returning `RequeueAfter` every time creates a loop forever.

## 3. Requeue Immediately

```go
return ctrl.Result{Requeue: true}, nil
```

If there is no error, `Requeue` is true, and `RequeueAfter` is not set, controller-runtime requeues the request using rate limiting.

This also uses backoff behavior.

Use this when another reconcile should happen soon, but there was not exactly an error.

In many controllers, `RequeueAfter` is clearer than `Requeue: true` because it makes the delay explicit.

## 4. Stop Reconciliation

```go
return ctrl.Result{}, nil
```

If there is no error and no requeue option, controller-runtime considers the reconcile complete.

The controller will not reconcile this object again until another event happens, such as:

- The custom resource changes
- A watched child resource changes
- The controller restarts and resyncs
- Another watched event maps to this object

This is the normal return when the actual state already matches the desired state.

## Quick Reference

| Return | Meaning |
| --- | --- |
| `return ctrl.Result{}, err` | Retry with rate limiting/backoff |
| `return ctrl.Result{RequeueAfter: d}, nil` | Retry after fixed delay |
| `return ctrl.Result{Requeue: true}, nil` | Retry with rate limiting/backoff |
| `return ctrl.Result{}, nil` | Done until another watch event happens |

## Common Pattern

```go
if err := r.Get(ctx, req.NamespacedName, ec2Instance); err != nil {
    return ctrl.Result{}, client.IgnoreNotFound(err)
}

if externalResourceIsStillCreating {
    return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}

if err := r.Update(ctx, ec2Instance); err != nil {
    return ctrl.Result{}, err
}

return ctrl.Result{}, nil
```

The main idea: return an error for unexpected failures, `RequeueAfter` for intentional polling, and empty result with `nil` when reconciliation is complete.

