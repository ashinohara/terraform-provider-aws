package aws

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/hashicorp/terraform/helper/acctest"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/terraform"
)

func TestAccAWSBatchJobQueue(t *testing.T) {
	var jq batch.JobQueueDetail
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccBatchJobQueueBasic, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     testAccBatchJobQueuePreCheck(t),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBatchJobQueueDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBatchJobQueueExists("aws_batch_job_queue.test_queue", &jq),
					testAccCheckBatchJobQueueAttributes(&jq),
				),
			},
		},
	})
}

func TestAccAWSBatchJobQueueUpdate(t *testing.T) {
	var jq batch.JobQueueDetail
	ri := acctest.RandInt()
	config := fmt.Sprintf(testAccBatchJobQueueBasic, ri)
	updateConfig := fmt.Sprintf(testAccBatchJobQueueUpdate, ri)
	resource.Test(t, resource.TestCase{
		PreCheck:     testAccBatchJobQueuePreCheck(t),
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckBatchJobQueueDestroy,
		Steps: []resource.TestStep{
			{
				Config: config,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBatchJobQueueExists("aws_batch_job_queue.test_queue", &jq),
					testAccCheckBatchJobQueueAttributes(&jq),
				),
			},
			{
				Config: updateConfig,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckBatchJobQueueExists("aws_batch_job_queue.test_queue", &jq),
					testAccCheckBatchJobQueueAttributes(&jq),
				),
			},
		},
	})
}

func testAccBatchJobQueuePreCheck(t *testing.T) func() {
	return func() {
		testAccPreCheck(t)
		if os.Getenv("AWS_ACCOUNT_ID") == "" {
			t.Fatal("AWS_ACCOUNT_ID must be set")
		}
	}
}

func testAccCheckBatchJobQueueExists(n string, jq *batch.JobQueueDetail) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		rs, ok := s.RootModule().Resources[n]
		log.Printf("State: %#v", s.RootModule().Resources)
		if !ok {
			return fmt.Errorf("Not found: %s", n)
		}

		if rs.Primary.ID == "" {
			return fmt.Errorf("No Batch Job Queue ID is set")
		}

		conn := testAccProvider.Meta().(*AWSClient).batchconn
		name := rs.Primary.Attributes["name"]
		queue, err := getJobQueue(conn, name)
		if err != nil {
			return err
		}
		if queue == nil {
			return fmt.Errorf("Not found: %s", n)
		}
		*jq = *queue

		return nil
	}
}

func testAccCheckBatchJobQueueAttributes(jq *batch.JobQueueDetail) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if !strings.HasPrefix(*jq.JobQueueName, "tf_acctest_batch_job_queue") {
			return fmt.Errorf("Bad Job Queue name: %s", *jq.JobQueueName)
		}
		for _, rs := range s.RootModule().Resources {
			if rs.Type != "aws_batch_job_queue" {
				continue
			}
			if *jq.JobQueueArn != rs.Primary.Attributes["arn"] {
				return fmt.Errorf("Bad Job Queue ARN\n\t expected: %s\n\tgot: %s\n", rs.Primary.Attributes["arn"], *jq.JobQueueArn)
			}
			if *jq.State != rs.Primary.Attributes["state"] {
				return fmt.Errorf("Bad Job Queue State\n\t expected: %s\n\tgot: %s\n", rs.Primary.Attributes["state"], *jq.State)
			}
			priority, err := strconv.ParseInt(rs.Primary.Attributes["priority"], 10, 64)
			if err != nil {
				return err
			}
			if *jq.Priority != priority {
				return fmt.Errorf("Bad Job Queue Priority\n\t expected: %s\n\tgot: %s\n", rs.Primary.Attributes["priority"], *jq.Priority)
			}
		}
		return nil
	}
}

func testAccCheckBatchJobQueueDestroy(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "aws_batch_job_queue" {
			continue
		}
		conn := testAccProvider.Meta().(*AWSClient).batchconn
		jq, err := getJobQueue(conn, rs.Primary.Attributes["name"])
		if err == nil {
			if jq != nil {
				return fmt.Errorf("Error: Job Queue still exists")
			}
		}
		return nil
	}
	return nil
}

const testAccBatchJobQueueBaseConfig = `
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
  role = "${aws_iam_role.batch_compute_environment.id}"
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

var testAccBatchJobQueueBasic = testAccBatchJobQueueBaseConfig + `
resource "aws_batch_job_queue" "test_queue" {
  name = "tf_acctest_batch_job_queue_%[1]d"
  state = "ENABLED"
  priority = 1
  compute_environments = ["${aws_batch_compute_environment.test_environment.arn}"]
}`

var testAccBatchJobQueueUpdate = testAccBatchJobQueueBaseConfig + `
resource "aws_batch_job_queue" "test_queue" {
  name = "tf_acctest_batch_job_queue_%[1]d"
  state = "DISABLED"
  priority = 2
  compute_environments = ["${aws_batch_compute_environment.test_environment.arn}"]
}`
