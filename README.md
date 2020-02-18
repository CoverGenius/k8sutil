# k8sutil
This is a command line tool with subcommands for creating and interpreting kubernetes objects.

## How to use
1. Clone this repo
2. cd into the newly created k8sutil subdirectory
3. `go build`
4. Run `./k8sutil --help` to show the available subcommands

## Available Subcommands
### Display the current context, current namespace, and current cluster configured by your Kubernetes Config File
`k8sutil get-context` will read from the `KUBECONFIG` environment variable if it is set, otherwise it is read from `~/.kube/config`.

![](screenshots/get-context_example_run)

### Easily switch current context and namespace for `kubectl`
TODO: Documentation
### Summarise Kubernetes Resource Information (remote or defined locally in YAMLs)
TODO: Documentation
### Lint YAML Kubernetes Resource Definitions for Security Vulnerabilities
TODO: Documentation
### Show Dependencies implied by YAML Kubernetes Resources
TODO: Documentation


### TODO (Future Work)
- The utils package is kind of a mess, want to make subdirectories based on each subcommand?
- I eventually want to make the linter extendable. You should be able to progamatically invoke it instead of just as a command-line tool, and you should be able to add your own custom requirements. I like this a lot because since my tool relies onfreehand boolean functions, your tests can literally be whatever you want. You aren't restricted to just set, equal, greaterthan field checks like in kube-lint. This would be really nice. There is a lot more flexibility with this. For example, you could check that a string field belongs to a collection of custom defined strings in your program. Maybe there's not much of a use case for it, but at least the option is there.
    - For this, I would need to finally resign myself to the fact that I will need different rule types for different resource type so I can defer the injection of the relevant resource. (Right now, when I instantiate a rule struct, I am relying on the fact that there is a resource pointer in scope, and this is just not flexible enough. I thought it was a cool idea at first, but I was a little bit wrong)
    - I can make separate types based on Resource type, eg DeploymentRule. It has a member function Condition with 1 `*appsv1.Deployment` parameter. Then from this, as soon as I get the reference to the deployment, I can create a Rule struct, so that all <Resource>rules will conform to the same structure and I can execute them all in one go and apply all the tests and fixes in a uniform way.
    - Would prefer to pull TypeMeta and ObjectMeta interface conformance tests right to the beginning when I first parse the yamls

