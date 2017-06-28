package aws

import (
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/batch"
	"github.com/hashicorp/terraform/helper/resource"
	"github.com/hashicorp/terraform/helper/schema"
	"github.com/hashicorp/terraform/helper/validation"
)

func resourceAwsBatchComputeEnvironment() *schema.Resource {
	return &schema.Resource{
		Create: resourceAwsBatchComputeEnvironmentCreate,
		Read:   resourceAwsBatchComputeEnvironmentRead,
		Update: resourceAwsBatchComputeEnvironmentUpdate,
		Delete: resourceAwsBatchComputeEnvironmentDelete,

		Schema: map[string]*schema.Schema{
			"name": {
				Type:     schema.TypeString,
				Required: true,
				ForceNew: true,
			},

			"compute_resources": {
				Type:     schema.TypeList,
				Required: true,
				MaxItems: 1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"bid_percentage": {
							Type:     schema.TypeInt,
							Optional: true,
							ForceNew: true,
						},
						"desired_vcpus": {
							Type:     schema.TypeInt,
							Optional: true,
						},
						"ec2_key_pair": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"image_id": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"instance_role_arn": {
							Type:         schema.TypeString,
							Required:     true,
							ValidateFunc: validateArn,
							ForceNew:     true,
						},
						"instance_types": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
							ForceNew: true,
						},
						"max_vcpus": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"min_vcpus": {
							Type:     schema.TypeInt,
							Required: true,
						},
						"security_group_ids": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
							ForceNew: true,
						},
						"spot_iam_fleet_role_arn": {
							Type:     schema.TypeString,
							Optional: true,
							ForceNew: true,
						},
						"subnets": {
							Type:     schema.TypeSet,
							Required: true,
							Elem:     &schema.Schema{Type: schema.TypeString},
							Set:      schema.HashString,
							ForceNew: true,
						},
						"tags": {
							Type:     schema.TypeMap,
							Optional: true,
							ForceNew: true,
						},
						"type": {
							Type:         schema.TypeString,
							Required:     true,
							ForceNew:     true,
							ValidateFunc: validation.StringInSlice([]string{batch.CRTypeEc2, batch.CRTypeSpot}, true),
						},
					},
				},
			},

			"service_role_arn": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validateArn,
			},

			"state": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{batch.CEStateEnabled, batch.CEStateDisabled}, true),
			},

			"type": {
				Type:         schema.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validation.StringInSlice([]string{batch.CETypeManaged, batch.CETypeUnmanaged}, true),
			},

			"arn": {
				Type:     schema.TypeString,
				Optional: true,
				Computed: true,
			},
		},
	}
}

func resourceAwsBatchComputeEnvironmentCreate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).batchconn
	input := batch.CreateComputeEnvironmentInput{
		ComputeEnvironmentName: aws.String(d.Get("name").(string)),
		ServiceRole:            aws.String(d.Get("service_role_arn").(string)),
		State:                  aws.String(d.Get("state").(string)),
		Type:                   aws.String(d.Get("type").(string)),
	}
	input.ComputeResources = createComputeResource(d)
	out, err := conn.CreateComputeEnvironment(&input)
	name := d.Get("name").(string)
	if err != nil {
		return fmt.Errorf("%s %q", err, name)
	}

	log.Println(
		"[INFO] Waiting for ComputeEnvironment to be available")

	stateConf := &resource.StateChangeConf{
		Pending:    []string{batch.CEStatusCreating, batch.CEStatusUpdating},
		Target:     []string{batch.CEStatusValid},
		Refresh:    batchComputeEnvironmentRefreshFunc(conn, name),
		Timeout:    10 * time.Minute,
		MinTimeout: 10 * time.Second,
		Delay:      3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf("[WARN] Error waiting for ComputeEnvironment state to be \"VALID\": %s", err)
	}

	arn := *out.ComputeEnvironmentArn
	log.Printf("[DEBUG] ComputeEnvironment created: %s", arn)
	d.SetId(arn)

	return resourceAwsBatchComputeEnvironmentRead(d, meta)
}

func resourceAwsBatchComputeEnvironmentRead(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).batchconn

	ce, err := getComputeEnvironment(conn, d.Get("name").(string))
	if err != nil {
		return err
	}
	if ce == nil {
		return fmt.Errorf("[WARN] Error reading Compute Environment: \"%s\"", err)
	}
	d.Set("arn", ce.ComputeEnvironmentArn)
	d.Set("compute_resources", flattenComputeResource(ce.ComputeResources))
	d.Set("name", ce.ComputeEnvironmentName)
	d.Set("service_role_arn", ce.ServiceRole)
	d.Set("state", ce.State)
	d.Set("type", ce.Type)
	return nil
}

func resourceAwsBatchComputeEnvironmentUpdate(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).batchconn

	updateInput := &batch.UpdateComputeEnvironmentInput{
		ComputeEnvironment: aws.String(d.Get("name").(string)),
		ComputeResources:   updateComputeResource(d),
		ServiceRole:        aws.String(d.Get("service_role_arn").(string)),
		State:              aws.String(d.Get("state").(string)),
	}
	_, err := conn.UpdateComputeEnvironment(updateInput)
	if err != nil {
		return err
	}
	return resourceAwsBatchComputeEnvironmentRead(d, meta)
}

func resourceAwsBatchComputeEnvironmentDelete(d *schema.ResourceData, meta interface{}) error {
	conn := meta.(*AWSClient).batchconn
	sn := d.Get("name").(string)

	// Check to make compute environment exists
	ce, err := getComputeEnvironment(conn, sn)
	if err != nil {
		return err
	}

	if ce == nil {
		log.Printf("[DEBUG] Compute Environment %q is already gone", sn)
		return err
	}

	_, err = conn.UpdateComputeEnvironment(&batch.UpdateComputeEnvironmentInput{
		ComputeEnvironment: aws.String(sn),
		State:              aws.String(batch.CEStateDisabled),
	})
	if err != nil {
		return err
	}

	// Wait until the Compute Environment is disabled
	err = resource.Retry(5*time.Minute, func() *resource.RetryError {
		log.Printf("[DEBUG] Trying to delete Compute Environment service %s", sn)
		_, err = conn.DeleteComputeEnvironment(&batch.DeleteComputeEnvironmentInput{
			ComputeEnvironment: aws.String(sn),
		})
		if err == nil {
			return nil
		}
		return resource.RetryableError(err)
	})

	stateConf := &resource.StateChangeConf{
		Pending:    []string{batch.CEStatusUpdating, batch.CEStatusDeleting},
		Target:     []string{batch.CEStatusDeleted},
		Refresh:    batchComputeEnvironmentRefreshFunc(conn, sn),
		Timeout:    10 * time.Minute,
		Delay:      10 * time.Second,
		MinTimeout: 3 * time.Second,
	}

	_, err = stateConf.WaitForState()
	if err != nil {
		return fmt.Errorf(
			"Error waiting for Compute Environment (%s) to be destroyed: %s",
			sn, err)
	}

	d.SetId("")
	return nil
}

func createComputeResource(d *schema.ResourceData) *batch.ComputeResource {
	data := d.Get("compute_resources").([]interface{})[0]
	compute := data.(map[string]interface{})
	resource := &batch.ComputeResource{
		BidPercentage:    aws.Int64(int64(compute["bid_percentage"].(int))),
		DesiredvCpus:     aws.Int64(int64(compute["desired_vcpus"].(int))),
		Ec2KeyPair:       aws.String(compute["ec2_key_pair"].(string)),
		ImageId:          aws.String(compute["image_id"].(string)),
		InstanceRole:     aws.String(compute["instance_role_arn"].(string)),
		MaxvCpus:         aws.Int64(int64(compute["max_vcpus"].(int))),
		MinvCpus:         aws.Int64(int64(compute["min_vcpus"].(int))),
		SpotIamFleetRole: aws.String(compute["spot_iam_fleet_role_arn"].(string)),
		Type:             aws.String(compute["type"].(string)),
	}

	if v, ok := compute["instance_types"]; ok && v.(*schema.Set).Len() > 0 {
		resource.InstanceTypes = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := compute["security_group_ids"]; ok && v.(*schema.Set).Len() > 0 {
		resource.SecurityGroupIds = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := compute["subnets"]; ok && v.(*schema.Set).Len() > 0 {
		resource.Subnets = expandStringList(v.(*schema.Set).List())
	}

	if v, ok := compute["tags"]; ok {
		resource.Tags = tagsFromMapGeneric(v.(map[string]interface{}))
	}

	return resource
}

func updateComputeResource(d *schema.ResourceData) *batch.ComputeResourceUpdate {
	data := d.Get("compute_resources").([]interface{})[0]
	compute := data.(map[string]interface{})
	resource := &batch.ComputeResourceUpdate{
		DesiredvCpus: aws.Int64(int64(compute["desired_vcpus"].(int))),
		MaxvCpus:     aws.Int64(int64(compute["max_vcpus"].(int))),
		MinvCpus:     aws.Int64(int64(compute["min_vcpus"].(int))),
	}

	return resource
}

func getComputeEnvironment(conn *batch.Batch, sn string) (*batch.ComputeEnvironmentDetail, error) {
	describeOpts := &batch.DescribeComputeEnvironmentsInput{
		ComputeEnvironments: []*string{aws.String(sn)},
	}
	resp, err := conn.DescribeComputeEnvironments(describeOpts)
	if err != nil {
		return nil, err
	}

	numComputeEnvs := len(resp.ComputeEnvironments)
	switch {
	case numComputeEnvs == 0:
		log.Printf("[DEBUG] Compute Environment %q is already gone", sn)
		return nil, nil
	case numComputeEnvs == 1:
		return resp.ComputeEnvironments[0], nil
	case numComputeEnvs > 1:
		return nil, fmt.Errorf("Multiple Compute Environments with name %s", sn)
	}
	return nil, nil
}

func batchComputeEnvironmentRefreshFunc(conn *batch.Batch, sn string) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		ce, err := getComputeEnvironment(conn, sn)
		if err != nil {
			return nil, "failed", err
		}
		if ce == nil {
			return 42, batch.CEStatusDeleted, nil
		}
		return ce, *ce.Status, nil
	}
}

func flattenComputeResource(cr *batch.ComputeResource) []map[string]interface{} {
	resource := make(map[string]interface{})
	resource["bid_percentage"] = cr.BidPercentage
	resource["desired_vcpus"] = cr.DesiredvCpus
	resource["ec2_key_pair"] = cr.Ec2KeyPair
	resource["image_id"] = cr.ImageId
	resource["instance_role_arn"] = cr.InstanceRole
	resource["instance_types"] = cr.InstanceTypes
	resource["max_vcpus"] = cr.MaxvCpus
	resource["min_vcpus"] = cr.MinvCpus
	resource["security_group_ids"] = cr.SecurityGroupIds
	resource["spot_iam_fleet_role_arn"] = cr.SpotIamFleetRole
	resource["subnets"] = cr.Subnets
	resource["type"] = cr.Type
	return []map[string]interface{}{resource}
}
