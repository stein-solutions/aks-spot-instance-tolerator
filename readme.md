# aks-spot-instance-tolerator

This tool allows the execution of most workloads on aks (Azure Kubernetes Service) spot instances. This is useful if your aks cluster only consists of interruptable workloads.

With spot instances you can **save up to 88% of compute cost** over on demand instances in AKS. With **aks-spot-instance-tolerator** it is actually **easy to deploy workloads on spot instances**. 

## The Problem

For a pod to be allowed to run on a spot instance node in aks, the pod needs to have the toleration `kubernetes.azure.com/scalesetpriority=spot:NoSchedule` or it will not be deployed on that node. To manually add this toleration to every Deployment, Statefulset, Job and pod (spawned through other means) is cumbersome and in some instances impossible. 

## The Solutions

The aks-spot-instance-tolerator automatically adds this toleration to every pod that is applied to the cluster, regardless of its origin. 

## How to install

The aks-spot-instance-tolerator can be installed through helm. 

1. run `helm repo add stein.solutions https://stein-solutions.github.io/helm-charts/`
2. run `helm install <release-name> stein-solutions/aks-spot-instance-tolerator`

## stein.solutions

You are aiming to save costs of your kubernetes clusters? You are looking for a strong partner to build your companies cloud platform? Get in touch: stein.solutions
