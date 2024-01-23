# Kubectl Copy-Secret Plugin

This is a kubectl plugin to simplify copying secrets from one namespace to another. There are likely much better ways to achieve the same result that would be recommended over this approach. If for any reason you don't want to go down those paths then this may be a good fit for you.

## What does this replace
If you've ever had to copy a secret from one namespace to another you may have used a script like the following

```shell
kubectl -n <origin-namespace> get secret <some-secret> -o json \
        | jq 'del(.metadata["namespace", "creationTimestamp","resourceVersion","selfLink","uid"])' \
        | kubectl apply -n <destination-namespace> -f -
```

This works really well! The downside is that you have to remember the particulars of the secret object plus both kubectl commands and jq. Depending on how much you work with these tools that may hard to remember when you need it. One option is to write a shell script to handle it for you. A simple shell script is easy enough, but you're still relying on these other tools. If your system is set up and stable this shouldn't be a problem, but if you move between systems then it's possible to run into some portability issues.

## Motivation
I created this plugin to scratch my own itch. I run a small single node cluster that only I admin and use for learning purposes. I have a number of secrets that I share between namespaces, the primary example being `imagePullSecrets` used for pulling container images from private docker registries. When I create a new namespace it's often (but not always) the case that I want to copy this secret to the new namespace.

In the past I've used `kubectl` plus `jq` like above to do the same thing, but this seemed like a good learning opportunity for golang and the kubernetes `client-go` library.

