# kube-ssm-secrets
The `kube-ssm-secrets` is a lightweight Kubernetes operator that creates and manages Kubernetes secrets using data from AWS Systems Manager (SSM) Parameter Store.
This operator watches for ServiceAccount resources with specific annotations, and creates/updates secrets based on the information provided.

## Features
- Automatically creates or updates Kubernetes secrets from AWS SSM Parameter Store using ServiceAccount annotations
- Supports single-value and YAML-formatted SSM parameters
- Reconciles and refreshes the secrets at a configurable interval
- Does not delete any Kubernetes Secrets after its creation, ensuring data persistence even if the operator is removed or stops working

## Why use annotations instead of CRDs
Using annotations on existing Kubernetes resources, like ServiceAccounts, provides a simple and non-intrusive way to create and manage Kubernetes Secrets. This approach has several advantages over classical ones like using Custom Resource Definitions (CRDs):

- **Ease of use**: `kube-ssm-secrets` uses annotations on ServiceAccount resources, which are simple and easy to understand. This makes it more accessible to developers and operators who may not be familiar with custom resources or complex configurations.
- **Non-intrusive**: By leveraging annotations, kube-ssm-secrets does not require any modification to existing Kubernetes deployments, making it easy to adopt in existing pipelines and processes without making significant changes.
- **Better integration with Helm**: As many Helm charts already provide options for applying custom annotations, `kube-ssm-secrets` can be easily integrated into existing Helm deployments without requiring changes to the charts themselves. This enables seamless management of secrets for applications deployed with Helm.
- **Reduced complexity**: kube-ssm-secrets avoids the need for additional custom resources or APIs, reducing the overall complexity of the solution and making it easier to maintain and troubleshoot.

## Usage
### Deploy the operator
Deploy the `kube-ssm-secrets` operator in your Kubernetes cluster using [chart](./chart/).
Here is the basic example:
```sh
helm install kube-ssm-secrets ./chart \
  --set aws.credentials.accessKey="${AWS_ACCESS_KEY_ID}" \
  --set aws.credentials.secretKey="${AWS_SECRET_ACCESS_KEY}" \
  --set aws.region="${AWS_DEFAULT_REGION}"
```

### Annotate your ServiceAccount resources
Annotate your ServiceAccount resources with the `secret.ssm-parameter/` prefix followed by the desired secret name. The annotation value must contain the SSM parameter path(s), and a key for each SSM parameter:
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-service-account
  annotations:
    secret.ssm-parameter/my-secret: "key1=/myapp/ssm/key1,key2=/myapp/ssm/key2"
```

For YAML-formatted SSM parameters, use a single path without a key:
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: my-service-account
  namespace: my-namespace
  annotations:
    secret.ssm-parameter/my-secret: "/myapp/ssm/yaml"
```

### Monitor created secrets
The operator will create or update the secrets specified in the ServiceAccount annotations. The secrets will be labeled with `app.kubernetes.io/managed-by=kube-ssm-secrets`:

    $ kubectl get secrets -A -l app.kubernetes.io/managed-by=kube-ssm-secrets -w

The operator will refresh the secrets at the configured interval.

## Authentication
`kube-ssm-secrets` uses the AWS SDK to interact with AWS Systems Manager Parameter Store. It requires appropriate permissions to read the SSM parameters. There are two common ways to provide these permissions:

- **IAM Roles for Service Accounts (IRSA)**: If you are running your Kubernetes cluster on Amazon EKS, you can use IAM Roles for Service Accounts to associate an IAM role with a Kubernetes service account. This way, any pod running with that service account will have the permissions granted by the IAM role.
- **Helm variables:** You can pass the required AWS credentials directly using Helm variables, please refer [chart](./chart/).

### To use IRSA, follow these steps:
1. Create an IAM policy granting the required permissions to access the SSM parameters:
```json
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": ["ssm:GetParameter*"],
            "Resource": "*"
        }
    ]
}
```
1. Create an IAM role and attach the policy created in previous step
1. Annotate the Kubernetes service account with the ARN of the IAM role by using values in Helm Chart or manually adding:
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kube-ssm-secrets
  annotations:
    eks.amazonaws.com/role-arn: arn:aws:iam::<AWS_ACCOUNT_ID>:role/kube-ssm-secrets-role
```
By using either IRSA or Helm variables, you can securely authenticate `kube-ssm-secrets` with AWS and provide the necessary permissions to access the SSM Parameter Store.

## License
`kube-ssm-secrets` is licensed under the Apache License, Version 2.0. See the [LICENSE](./LICENSE) file for details.
