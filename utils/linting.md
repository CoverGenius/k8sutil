### What this linter does

This linter enforces kubernetes best practices. [Here](https://thenewstack.io/10-kubernetes-best-practices-you-can-easily-apply-to-your-clusters/) 
is an example of popular best practices.  This tool is a subcommand of `service` and you can provide one or more files or 
folders to lint as argument to the subcommand `lint-k8s`. The arguments you pass are treated as one cohesive unit to be analysed if you do not specify `--standalone-mode`.

Refer to [linter_usage.md](./linter_usage.md) for more information on how to use the command.

### Rules enforced by the linter
#### Independent Rules
A Container:

   - must have a security context key present
   - must disallow privilege escalation
   - must not be privileged
   - must have resource limits and requests set
   - must not request more than 1 unit of CPU
   - must use an image from a set of allowed registries

 A Pod:

   - must have a security context key present
   - must run as non root
   - must enforce that its containers have user and group ID set to 44444 
   - must define exactly one container
   - must have its container comply with Container rules
   
 A Deployment:
 
   - must have a `project` label set
   - must have an `app.kubernetes.io/name` label set
   - must be within a namespace
   - Pod placement to prevent eviction (TBD)
   - must define a liveness and readiness probe for its container
   - must have its PodSpec comply with Pod rules
   - should not have matching readiness and liveness endpoints 
   - Must be an `apps/v1` Deployment (ref: `deprecated_apiversions.go`)

A CronJob:

   - must be within a namespace
   - must disallow concurrent operations
   - must have the security context set as above (TBD)
   - must have resource consumption set (TBD)
   - must have its PodSpec comply with Pod rules

A Job:

  - must be within a namespace
  - must have security context set
  - must have resource consumption set
  - must have cleanup policy
  - must use an image from a set of allowed registries
   - must have its PodSpec comply with Pod rules

A Namespace:

  - must have a name that is a valid DNS

A Service:

  - must be within a namespace
  - name must be a valid DNS

A Network Policy:
  
  - must be a `networking.k8s.io/v1` Network Policy (ref: `deprecated_apiversions.go`)

You can find the implementation of all of these rules within `lowerCasedResourceName.go`. 
For example, you will find rules relating to a Deployment object within `deployment.go`.

There are also rules that relate to the state of the entire unit of analysis. 

#### Interdependent Rules
- Every resource must be within the namespace present in the unit
- The unit should only contain one service, if any
- The unit must contain exactly one namespace object
- There must be a network policy defined for this namespace

You can find the implementation of these rules in `interdependentchecks.go`.

### Order of Evaluation of the Rules

Sometimes, the semantics of one rule relies on the fact that another rule has been satisfied. For example, a rule A could test whether a deployment has a particular key present in the `securityContext` field. But for this to make sense, the `securityContext` key must be present (rule B). Therefore, rule B is a prerequisite of rule A. The rules are tested in topologically sorted order thanks to `RuleSorter` to ensure that the dependent rules make sense to even execute at all. If a prerequisite test fails, all dependent tests are not executed, but will be reported to the user, since it is impossible to satisfy them, and it seems sensible to let the user know what they will need to do in order to avoid the error on a later pass.

#### How to use this feature when implementing your own rule
If the body of your rule performs some pointer or slice dereference for example, and you notice that many rules perform this dereference, it would be good if there was a way to avoid checking whether the pointer is non-nil or the slice is of the appropriate length every single time. This is a good opportunity to factor out the length check or nil check into a separate rule, and to add this rule to the list of prerequisites for any rule that performs the dereference. For example, imagine you write 5 rules about the semantics of a deployment's container, and in every single rule body, you need to check that `len(containers) == 1`. It would be better to write a separate test that only checks the length of `containers` called `DEPLOYMENT_EXACTLY_1_CONTAINER` and list this as a prerequisite for any follow-up tests, for example, a test that checks whether `deployment.Spec.Template.Spec.Containers[0].runAsNonRoot == true`.

### Limitations

There are no independent rules set up specifically for an Ingress, Network Policy, Persistent Volume Claim, Role, 
Role Binding, or Service Account yet.

The tool doesn't currently support rules that involve more than two resources. 
This would require considering how a linter message for such a rule would be printed. 
It would need to be implemented in `lint-k8s.go/LinterMessage`.

It would also help if the construction of the Message field was lazy, so that we could interpolate useful information into the error messages
when the test fails. This might require wrapping the string in a function and this could be a future extension of the linter.

### How to add a new rule to the linter

Any `lowerCasedResourceName.go` file or `interdependentchecks.go` contain an exported `<Resource>Rules` function that returns a list of linter `Rule`s. 
You need to either append a rule struct manually to the list that is returned, or manually add the new linter rule struct as a literal 
within the list of rule structs that is either returned or appended within the function. You should specify the key `Condition` to be a 
boolean function which, when evaluated to false, will cause the message in the field `Message` to appear in the error log. 
You need to specify a level in the `Level` field of the error, either `WARNING` or `ERROR`. In order to interpolate relevant fields 
of the resource into the error message, it is important that you pass a non-empty slice of `YamlDerivedKubernetesResource`s to the `Resources` 
key. For an independent rule, this is always just the parameter of the function. For an interdependent rule, you need to decide which resources 
are relevant to the error, and pass references to those resources into the list. You also need to describe how to fix the error if it fails 
within the Fixfield. This is a boolean function, returning false if the error could not be fixed. Therefore, if the error cannot be fixed, you 
should just leave this as a boolean function returning false. If you can fix it, you should modify the resource in-place somehow and return true. 
This will help ensure that rules which are dependent on this rule being successful can still be checked and fixed themselves. If you can 
successfully apply a fix in some cases, you should also specify the `FixDescription` field so that if the user specifies the flag `--report-fix`, 
the description of what was successfully fixed in this case will appear in the list of fixes also. Otherwise, there will be an empty string in 
this list and this may worry the user.

### Example Walkthrough

Maybe, we now want to enforce that the root file system is read only. This rule relates to a Deployment object.

 1. Find the relevant file to add the linter rule to. In this case, it is `deployment.go`.
 2. Add a new `&Rule` literal to the `deploymentRules` slice. The `Condition` should check that the 
[`readOnlyRootFilesystem`](https://kubernetes.io/docs/concepts/policy/pod-security-policy/#volumes-and-file-systems) 
key is present, and it is set to true. To find out how this key is represented as an already-parsed golang object, refer
to the [documentation](#documentation).

```go
...
},
&Rule{
  // ??
},
```

 3. We see that a `Deployment` struct has a `DeploymentSpec` field. Within this, there is a `PodTemplateSpec` struct that has a 
`SecurityContext` field. This has a `ReadOnlyFileSystem *bool`. All we need to do is access the security context object, 
check if this field is `nil`, and if not, check that it is indeed true. The message that should be printed on error is 
`The File System must be read-only`, and the relevant resources are the `Deployment` and nothing else.

```go
...
},
&Rule{
  Condition: func() bool {
    return securityContext.ReadOnlyFileSystem != nil &&
         *securityContext.ReadOnlyFileSystem == true
  },
  Message: "The File System must be read-only",
  Level: ERROR,
  Resources: []*YamlDerivedKubernetesResource{resource},
},
```
The `securityContext` struct had already been stored in a local variable earlier so we can reuse this in the closure.

4. We can also implement a `Fix` function for this rule since setting the key to the correct value doesn't require further input.
We just need to create a local variable set to true and point the `ReadOnlyFileSystem` member to this variable. Also notice that 
I've included the security context being non-nil as a prerequisite for this test, since we dereference this field in the boolean
return statement. This will ensure that there's no way we can cause a runtime panic due to a nil dereference.

```go
...
},
&Rule{
  Prereqs: []RuleID{DEPLOYMENT_EXISTS_SECURITY_CONTEXT},
  Condition: func() bool {
    return securityContext.ReadOnlyFileSystem != nil &&
         *securityContext.ReadOnlyFileSystem == true
  },
  Message: "The File System must be read-only",
  Level: ERROR,
  Resources: []*YamlDerivedKubernetesResource{resource},
  Fix: func() bool {
     readOnly := true
     *securityContext.ReadOnlyFileSystem = &readOnly
     return true
  },
  FixDescription: fmt.Sprintf("Set %s's securityContext.readOnlyFileSystem to true", deployment.Name),
},
```

**How to find the relevant documentation for a particular resource**

Check which package contains the Resource. It will be obvious by the way the type is prefixed. For example, a 
`Deployment` is within the `appsv1` package. Refer to the import statement `appsv1 k8s.io/api/apps/v1`, 
This tells you that the documentation is at [https://godoc.org/k8s.io/api/apps/v1](https://godoc.org/k8s.io/api/apps/v1).
