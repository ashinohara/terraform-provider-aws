---
layout: "aws"
page_title: "AWS: batch"
sidebar_current: "docs-aws-resource-batch-compute-environment"
description: |-
  Provides a Batch Compute Environment resource.
---

# aws\_batch\_compute\_environment

-> **Note:** AWS Batch requires a specific IAM policy to be associated with the compute
    environment in order to function properly.  The `aws_iam_role_policy` shown below
    contains the correct policy for AWS batch.

-> **Note:** To prevent a race condition during environment deletion, make sure to set `depends_on` to the related `aws_iam_role_policy`;
    otherwise, the policy may be destroyed too soon and the compute environment will then get stuck in the `DELETING` state.

Provides a Batch Compute Environment resource.

## Example Usage

```hcl
resource "aws_batch_compute_environment" "test_environment" {
  name              = "tf-test-compute-environment"
  compute_resources = {
    instance_role_arn  = "${aws_iam_role.batch_compute_environment.arn}"
    instance_types     = ["m3.medium"]
    max_vcpus          = 1
    min_vcpus          = 0
    security_group_ids = ["${aws_security_group.bar.id}"]
    subnets            = ["${aws_vpc.foo.id}"]
    type               = "EC2"
  }
  service_role_arn  = "${aws_iam_role.batch_compute_environment.arn}"
  state             = "ENABLED"
  type              = "MANAGED"
  depends_on        = ["aws_iam_role_policy.batch_compute_environment_policy"]
}

resource "aws_security_group" "bar" {
    name = "tf-test-security-group-%[1]d"
    description = "tf-test-batch-compute-environment"
    vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_iam_role" "batch_compute_environment" {
  name = "tf-test-compute-environment-role"
  assume_role_policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "batch.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF
}

resource "aws_iam_role_policy" "batch_compute_environment_policy" {
  name = "tf-test-compute-environment-policy"
  role       = "${aws_iam_role.batch_compute_environment.id}"
  policy = <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeAccountAttributes",
        "ec2:DescribeInstances",
        "ec2:DescribeSubnets",
        "ec2:DescribeSecurityGroups",
        "ec2:DescribeKeyPairs",
        "ec2:DescribeImages",
        "ec2:DescribeImageAttribute",
        "ec2:DescribeSpotFleetInstances",
        "ec2:DescribeSpotFleetRequests",
        "ec2:DescribeSpotPriceHistory",
        "ec2:RequestSpotFleet",
        "ec2:CancelSpotFleetRequests",
        "ec2:ModifySpotFleetRequest",
        "ec2:TerminateInstances",
        "autoscaling:DescribeAccountLimits",
        "autoscaling:DescribeAutoScalingGroups",
        "autoscaling:DescribeLaunchConfigurations",
        "autoscaling:DescribeAutoScalingInstances",
        "autoscaling:CreateLaunchConfiguration",
        "autoscaling:CreateAutoScalingGroup",
        "autoscaling:UpdateAutoScalingGroup",
        "autoscaling:SetDesiredCapacity",
        "autoscaling:DeleteLaunchConfiguration",
        "autoscaling:DeleteAutoScalingGroup",
        "autoscaling:CreateOrUpdateTags",
        "autoscaling:SuspendProcesses",
        "autoscaling:PutNotificationConfiguration",
        "autoscaling:TerminateInstanceInAutoScalingGroup",
        "ecs:DescribeClusters",
        "ecs:DescribeContainerInstances",
        "ecs:DescribeTaskDefinition",
        "ecs:DescribeTasks",
        "ecs:ListClusters",
        "ecs:ListContainerInstances",
        "ecs:ListTaskDefinitionFamilies",
        "ecs:ListTaskDefinitions",
        "ecs:ListTasks",
        "ecs:CreateCluster",
        "ecs:DeleteCluster",
        "ecs:RegisterTaskDefinition",
        "ecs:DeregisterTaskDefinition",
        "ecs:RunTask",
        "ecs:StartTask",
        "ecs:StopTask",
        "ecs:UpdateContainerAgent",
        "ecs:DeregisterContainerInstance",
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents",
        "logs:DescribeLogGroups",
        "iam:GetInstanceProfile",
        "iam:PassRole"
      ],
      "Resource": "*"
    }
  ]
}
EOF
}
```

## Argument Reference

The following arguments are supported:

* `name` - (Required) Specifies the name of the compute environment.
* `compute_resources` - (Required) Details of the compute resources managed
    by the compute environment.  The maximum number of `compute_resources` blocks is `1`. Defined below.
* `service_role_arn` - (Required) Specifies the IAM role that allows
    AWS Batch to make calls to other AWS services on your behalf.
* `state` - (Optional) Specifies the state of the compute environment.
    If the state is ENABLED, then the compute environment accepts jobs from a queue.
    Must be one of: `ENABLED` or `DISABLED`
* `type` - (Optional) Specifies the type of the compute environment. Must be one of: `MANAGED` or `UNMANAGED`

## compute_resources

`compute_resources` supports the following:

* `bid_percentage` - (Optional) The minimum percentage that a Spot Instance price
    must be when compared with the On-Demand price for that instance type before instances are launched.
* `desired_vcpus` - (Optional) The desired number of EC2 vCPUS in the compute environment.
* `ec2_key_pair` - (Optional) The EC2 key pair that is used for instances launched in the compute environment.
* `image_id` - (Optional) The Amazon Machine Image (AMI) ID used for instances launched in the compute environment.
* `instance_role_arn` - (Required) The Amazon ECS instance role applied to Amazon EC2 instances in a compute environment.
* `instance_types` - (Required) The instances types that may launched.
* `max_vcpus` - (Required) The maximum number of EC2 vCPUs that an environment can reach.
* `min_vcpus` - (Required) The minimum number of EC2 vCPUs that an environment should maintain.
* `security_group_ids` - (Required) The EC2 security group that is associated with instances launched in the compute environment.
* `spot_iam_fleet_role_arn` - (Optional) The Amazon Resource Name (ARN)
    of the Amazon EC2 Spot Fleet IAM role applied to a SPOT compute environment.
* `subnets` - (Required) The VPC subnets into which the compute resources are launched.
* `tags` - (Optional) Key-value pair tags to be applied to resources that are launched in the compute environment.
* `type` - (Required) The type of compute environment. Must be one of: `EC2` or `SPOT`

## Attribute Reference

The following attributes are exported:

* `arn` - The Amazon Resource Name of the compute environment.
