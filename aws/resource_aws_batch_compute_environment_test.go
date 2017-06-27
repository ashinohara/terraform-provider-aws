package aws

import (
	"fmt"
	"log"
	"os"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSBatchComputeEnvironment(t *testing.T) {
	var computeEnv batch.ComputeEnvironmentDetail
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccBatchComputeEnvironmentBasic, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     testAccBatchComputeEnvironmentPreCheck(t),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBatchComputeEnvironmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBatchComputeEnvironmentExists("aws_batch_compute_environment.test_environment", &computeEnv),
					testAccCheckBatchComputeEnvironmentAttributes(&computeEnv, nil),
				),
			},
		},
	})
}

func TestAccAWSBatchComputeEnvironmentUpdate(t *testing.T) {
	var computeEnv batch.ComputeEnvironmentDetail
	log.SetOutput(os.Stdout)
	maxCpus := int64(2)
	minCpus := int64(1)
	computeResource := batch.ComputeResource{
		MaxvCpus:     &maxCpus,
		MinvCpus:     &minCpus,
		DesiredvCpus: &minCpus,
	}
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccBatchComputeEnvironmentBasic, ri)
	updateConfig := fmt.Sprintf(testAccBatchComputeEnvironmentUpdate, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     testAccBatchComputeEnvironmentPreCheck(t),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBatchComputeEnvironmentDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBatchComputeEnvironmentExists("aws_batch_compute_environment.test_environment", &computeEnv),
					testAccCheckBatchComputeEnvironmentAttributes(&computeEnv, nil),
				),
			},
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBatchComputeEnvironmentExists("aws_batch_compute_environment.test_environment", &computeEnv),
					testAccCheckBatchComputeEnvironmentAttributes(&computeEnv, &computeResource),
				),
			},
		},
	})
}

func testAccCheckBatchComputeEnvironmentExists(n string, computeEnv *batch.ComputeEnvironmentDetail) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		log.Printf("State: %#v", s.RootModule().Resources)
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Batch Compute Environment ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).batchconn
		name := rs.Primary.Attributes["name"]
		ce, err := getComputeEnvironment(conn, name)
		if err != nil {
			return err
		}
		if ce == nil {
			return fmt.Errorf("Not found: %s", n)
		}
		*computeEnv = *ce

		return nil
	}
}

func testAccBatchComputeEnvironmentPreCheck(t *testing.T) func() {
	return func() {
		testAccPreCheck(t)
		if os.Getenv("AWS_ACCOUNT_ID") == "" {
			t.Fatal("AWS_ACCOUNT_ID must be set")
		}
	}
}

func testAccCheckBatchComputeEnvironmentAttributes(computeEnv *batch.ComputeEnvironmentDetail, computeResource *batch.ComputeResource) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if !strings.HasPrefix(*computeEnv.ComputeEnvironmentName, "tf_acctest_batch_compute_environment") {
			return fmt.Errorf("Bad Compute Environment name: %s", *computeEnv.ComputeEnvironmentName)
		}
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_batch_compute_environment" {
				continue
			}
			if *computeEnv.ComputeEnvironmentArn != rs.Primary.Attributes["arn"] {
				return fmt.Errorf("Bad Compute Environment ARN\n\t expected: %s\n\tgot: %s\n", rs.Primary.Attributes["arn"], *computeEnv.ComputeEnvironmentArn)
			}
			if *computeEnv.State != rs.Primary.Attributes["state"] {
				return fmt.Errorf("Bad Compute Environment State\n\t expected: %s\n\tgot: %s\n", rs.Primary.Attributes["state"], *computeEnv.State)
			}

			if computeResource != nil {
				if *computeEnv.ComputeResources.MinvCpus != *computeResource.MinvCpus {
					return fmt.Errorf("Bad Compute Environment MinvCpus\n\t expected: %s\n\tgot: %s\n", *computeEnv.ComputeResources.MinvCpus, *computeResource.MinvCpus)
				}
				if *computeEnv.ComputeResources.MaxvCpus != *computeResource.MaxvCpus {
					return fmt.Errorf("Bad Compute Environment MaxvCpus\n\t expected: %s\n\tgot: %s\n", *computeEnv.ComputeResources.MaxvCpus, *computeResource.MaxvCpus)
				}
				if *computeEnv.ComputeResources.DesiredvCpus != *computeResource.DesiredvCpus {
					return fmt.Errorf("Bad Compute Environment DesiredvCpus\n\t expected: %s\n\tgot: %s\n", *computeEnv.ComputeResources.DesiredvCpus, *computeResource.DesiredvCpus)
				}
			}
		}
		return nil
	}
}

func testAccCheckBatchComputeEnvironmentDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_batch_compute_environment" {
			continue
		}
		conn := testAccProvider.Meta().(*AWSClient).batchconn
		describeOpts := &batch.DescribeComputeEnvironmentsInput{
			ComputeEnvironments: []*string{aws.String(rs.Primary.Attributes["name"])},
		}
		resp, err := conn.DescribeComputeEnvironments(describeOpts)
		if err == nil {
			if len(resp.ComputeEnvironments) != 0 {
				return fmt.Errorf("Error: Compute Environment still exists")
			}
		}
		return nil
	}
	return nil
}

const testAccBatchComputeEnvironmentBaseConfig = `
resource "aws_vpc" "foo" {
	cidr_block = "10.1.0.0/16"
	tags {
		Name = "testAccBatchComputeEnvironment"
	}
}

resource "aws_subnet" "foo" {
	cidr_block = "10.1.1.0/24"
	vpc_id = "${aws_vpc.foo.id}"
}

resource "aws_security_group" "bar" {
    name = "tf-test-security-group-%[1]d"
    description = "tf-test-batch-compute-environment"
    vpc_id = "${aws_vpc.foo.id}"
    ingress {
        from_port = -1
        to_port = -1
        protocol = "icmp"
        cidr_blocks = ["0.0.0.0/0"]
    }
}

resource "aws_iam_role" "batch_compute_environment" {
  name = "tf_acctest_batch_compute_environment_role_%[1]d"
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
  name = "tf-test-batch-compute-environment-policy_%[1]d"
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
}`

var testAccBatchComputeEnvironmentBasic = testAccBatchComputeEnvironmentBaseConfig + `
resource "aws_batch_compute_environment" "test_environment" {
  name = "tf_acctest_batch_compute_environment_%[1]d"
  compute_resources = {
    instance_role_arn = "${aws_iam_role.batch_compute_environment.arn}"
    instance_types = ["m3.medium"]
    max_vcpus = 1
    min_vcpus = 0
    security_group_ids = ["${aws_security_group.bar.id}"]
    subnets = ["${aws_vpc.foo.id}"]
    type = "EC2"
  }
  service_role_arn = "${aws_iam_role.batch_compute_environment.arn}"
  state = "ENABLED"
  type = "MANAGED"
  depends_on = ["aws_iam_role_policy.batch_compute_environment_policy"]
}`

var testAccBatchComputeEnvironmentUpdate = testAccBatchComputeEnvironmentBaseConfig + `
resource "aws_batch_compute_environment" "test_environment" {
  name = "tf_acctest_batch_compute_environment_%[1]d"
  compute_resources = {
    instance_role_arn = "${aws_iam_role.batch_compute_environment.arn}"
    instance_types = ["m3.medium"]
    max_vcpus = 2
    desired_vcpus = 1
    min_vcpus = 1
    security_group_ids = ["${aws_security_group.bar.id}"]
    subnets = ["${aws_vpc.foo.id}"]
    type = "EC2"
  }
  service_role_arn = "${aws_iam_role.batch_compute_environment.arn}"
  state = "DISABLED"
  type = "MANAGED"
  depends_on = ["aws_iam_role_policy.batch_compute_environment_policy"]
}`
