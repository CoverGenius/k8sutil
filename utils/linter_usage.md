## How to use the `k8sutil lint` linter

This is a reference for the command's usage.

```
Lint YAML file(s) against a set of predefined kubernetes best practices
Usage:
  xops service lint-k8s <filepath>* [flags]
Flags:
-d, --directories strings A comma-separated list of directories to recursively search for YAML documents
--fix apply fixes after identifying errors, where possible
-h, --help  help for lint-k8s
--fix-output string  output fixed yaml to file or folder instead of stdout
--standalone-mode Standalone mode - only run lint on the specified resources and skips any dependency checks
```

The `--out` argument can not only be a filepath, but it can also be a directory. In that case, the fixed `.yaml` filename will be based on the original `.yaml` filename with the suffix `.fixed` prepended to the `.yaml` extension. For example, if the file `deployment.yaml` is linted with `--fix-output` argument set to `myDir` and the `--fix` flag is set, then there will be a new file `myDir/deployment.fixed.yaml` once the linter has successfully completed. The directory needs to already exist for this to successfully execute.

When the  `--fix` flag is specified, the linter tries its best to autocorrect any field that does not require further input from the user. For example, it can set the user and group ID correctly without intervention. But if there is a required label missing, this requires further input because the label's value cannot be determined. Therefore, the error will remain in the fixed file for a human to manually fix. The linter error messages are intended to be informative enough so that the process of correcting the yaml file is straightforward.

### Invocations of the command and their meaning

(1) `xops service lint-k8s deployment.yaml --standalone-mode`
Analyse `deployment.yaml` based on the linter rules documented in `linting.md`, printing a failure or warning message when the linter rule is not satisfied. Do not test this resource against any interdependent rules.

**Example**

```
./xops service lint-k8s test_k8s_yaml/deployment_invalid_user_group_ids.yaml --standalone-mode
PASS - deployment_invalid_user_group_ids.yaml contains a valid Deployment
ERR - deployment_invalid_user_group_ids.yaml:1 (Deployment hello-world-web): The user and group ID should be set to 44444
```

(2) `xops service lint-k8s deployment.yaml --standalone-mode --fix`
Just like (1), but also applying any fixes where possible and printing the result to stdout.

(3) `xops service lint-k8s deployment.yaml --standalone-mode --fix --out fixed.yaml`
Just like (2), but printing the result to `$(pwd)/fixed.yaml` instead of stdout.

**Example**

```
$ ./xops service lint-k8s test_k8s_yaml/deployment_invalid_user_group_ids.yaml \
  --standalone-mode --fix --out fixed.yaml
$ diff fixed.yaml test_k8s_yaml/deployment_invalid_user_group_ids.yaml
63c63
< runAsGroup: 44444
---
> runAsGroup: 1000
65c65
< runAsUser: 44444
---
> runAsUser: 1000
$ ./xops service lint-k8s fixed.yaml --standalone-mode
PASS - fixed.yaml contains a valid Deployment
```
(4) `xops service -d partially_wrong_unit --fix --out .`
Lint all files within the `partially_wrong_unit` directory and output the fixed `.yaml` files to the current directory with the `.fixed` suffix.

**Example**
```
$ ./xops service lint-k8s \
		-d test_k8s_yaml/partially_wrong_unit_directory \
		--fix \
		--out test_k8s_yaml/partially_wrong_unit_directory
PASS - Deployment.yaml contains a valid Deployment
PASS - Namespace.yaml contains a valid Namespace
PASS - NetworkPolicy.yaml contains a valid NetworkPolicy
PASS - NetworkPolicy1.yaml contains a valid NetworkPolicy
PASS - Role.yaml contains a valid Role
PASS - RoleBinding.yaml contains a valid RoleBinding
PASS - Service.yaml contains a valid Service
PASS - ServiceAccount.yaml contains a valid ServiceAccount
ERR - Deployment.yaml:1 (Deployment hello-world-web): The user and group ID should be set to 44444
ERR - Deployment.yaml:1 (Deployment hello-world-web): There should be an app.kubernetes.io/name label present for 	the deployment's spec.template
ERR - Deployment.yaml:1 (Deployment hello-world-web): The image from this registry is not allowed. Expected an image from: []string{"277433404353.dkr.ecr.eu-central-1.amazonaws.com"}, Got image: "amitsaha/webapp-demo:golang-tls"
WARN - Deployment.yaml:1 (Deployment hello-world-web): It's recommended that the readiness and liveness probe endpoints don't match

$ ls test_k8s_yaml/partially_wrong_unit_directory
Deployment.fixed.yaml  
Namespace.yaml  
NetworkPolicy1.fixed.yaml  
Role.yaml  
Service.fixed.yaml  
ServiceAccount.yaml 
Deployment.yaml  
NetworkPolicy.fixed.yaml  
NetworkPolicy1.yaml  
RoleBinding.fixed.yaml  
Service.yaml 
Namespace.fixed.yaml  
NetworkPolicy.yaml  
Role.fixed.yaml  
RoleBinding.yaml  
ServiceAccount.fixed.yaml
```

(5) `xops service lint-k8s deployment.yaml service.yaml --standalone-mode --fix --out result.yaml`
Lint the two `.yaml` without applying interdependent checks, and write the fixed version of the two files to `result.yaml`. This will be a multi-document `.yaml` with a `---` delimiting the serialised form of each object.
