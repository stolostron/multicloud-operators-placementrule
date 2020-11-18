# Deployment Guide

<!-- START doctoc generated TOC please keep comment here to allow auto update -->
<!-- DON'T EDIT THIS SECTION, INSTEAD RE-RUN doctoc TO UPDATE -->
**Table of Contents**  *generated with [DocToc](https://github.com/thlorenz/doctoc)*

- [Deployment Guide](#deployment-guide)
    - [RBAC](#rbac)
        - [Deployment](#deployment)
    - [General process](#general-process)
<!-- END doctoc generated TOC please keep comment here to allow auto update -->

## RBAC

The service account is `multicluster-operators-placementrule`.

The role `multicluster-operators-placementrule` is binded to that service account.

### Deployment

```shell
cd multicloud-operators-placementrule
kubectl apply -f deploy/crds
kubectl apply -f deploy
```

## General process

Placementrule CR:

```yaml
apiVersion: apps.open-cluster-management.io/v1
kind: PlacementRule
metadata:
  name: all-ready-clusters
  namespace: default
spec:
  clusterSelector: {}
  clusterConditions:
    - type: ManagedClusterConditionAvailable
      status: "True"
```
