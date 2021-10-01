package test

import (
	"github.com/gruntwork-io/terratest/modules/logger"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/gruntwork-io/terratest/modules/testing"
	"github.com/stretchr/testify/assert"
	"path/filepath"
	"sort"
	gotesting "testing"
)

func getKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Get values of a map in the same order that getKeys gets the keys of the map,
// which is lexicographically sored by keys.
func getValues(m map[string]string) []string {
	values := make([]string, 0, len(m))
	keys := getKeys(m)
	for _, k := range keys {
		values = append(values, m[k])
	}
	return values
}

func assertValueStartsWith(t *gotesting.T, m map[string]string, rx interface{}) {
	for _, v := range m {
		assert.Regexp(t, rx, v)
	}
}

func assertTagFromResource(t *gotesting.T, plan *terraform.PlanStruct, address string, key string, value string) {
	terraform.RequirePlannedValuesMapKeyExists(t, plan, address)
	awsSubnet := plan.ResourcePlannedValuesMap[address]
	awsTagValue := awsSubnet.AttributeValues["tags"].(map[string]interface{})[key]
	assert.Equal(t, value, awsTagValue)
}

type discardLogger struct{}

func (_ discardLogger) Logf(_ testing.TestingT, format string, args ...interface{}) {}

// Test the Terraform module in examples/complete using Terratest.
func TestExamplesComplete(t *gotesting.T) {
	// Init phase module download fails when run in parallel
	//t.Parallel()

	terraformOptions := &terraform.Options{
		// The path to where our Terraform code is located
		TerraformDir: "../../examples/complete",
		Upgrade:      true,
		// Variables to pass to our Terraform code using -var-file options
		VarFiles: []string{"fixtures.us-east-2.tfvars"},
	}
	planLogger := logger.New(discardLogger{})
	terraformPlanOptions := &terraform.Options{
		TerraformDir: "../../examples/complete",
		Upgrade:      true,
		VarFiles:     []string{"fixtures.us-east-2.tfvars"},
		// Comment out this logger to get the json output
		Logger:       planLogger,
		PlanFilePath: filepath.Join("plan.out"),
	}

	// At the end of the test, run `terraform destroy` to clean up any resources that were created
	defer terraform.Destroy(t, terraformOptions)

	// This will run `terraform init`, `terraform plan`, and `terraform show` returning a struct
	plan := terraform.InitAndPlanAndShowWithStruct(t, terraformPlanOptions)

	// Validate the plan data
	var address string
	address = "module.private_only_subnets.aws_subnet.private[\"us-east-2a\"]"
	assertTagFromResource(t, plan, address, "Name", "eg-test-multi-az-subnets-private-ue2a")
	address = "module.private_subnets.aws_subnet.private[\"us-east-2a\"]"
	assertTagFromResource(t, plan, address, "Name", "eg-test-multi-az-subnets-private-use2a")
	address = "module.public_only_subnets.aws_subnet.public[\"us-east-2a\"]"
	assertTagFromResource(t, plan, address, "Name", "eg-test-multi-az-subnets-public-us-east-2a")
	address = "module.public_subnets.aws_subnet.public[\"us-east-2a\"]"
	assertTagFromResource(t, plan, address, "Name", "eg-test-multi-az-subnets-public-us-east-2a")

	// This will run `terraform apply` and fail the test if there are any errors.
	// If no errors it runs `terraform plan` and fail the test if there are additional changes.
	terraform.ApplyAndIdempotent(t, terraformOptions)

	/*
	   Outputs:

	   private_az_route_table_ids = {
	     "us-east-2a" = "rtb-0489137a5c668e49b"
	     "us-east-2b" = "rtb-083c0e942abb4b8a1"
	     "us-east-2c" = "rtb-0c36484693db5e774"
	   }
	   private_az_subnet_ids = {
	     "us-east-2a" = "subnet-0f56deccfe81c0ea0"
	     "us-east-2b" = "subnet-05861d30d45e7b675"
	     "us-east-2c" = "subnet-036d747a2b46857ae"
	   }
	   private_az_subnet_cidr_blocks = {
	     "us-east-2a" = "172.16.128.0/21"
	     "us-east-2b" = "172.16.136.0/21"
	     "us-east-2c" = "172.16.144.0/21"
	   }
	   public_az_ngw_ids = {
	     "us-east-2a" = "nat-0f5057f09b8cd8ddc"
	     "us-east-2b" = "nat-0971b2505ea6d03f1"
	     "us-east-2c" = "nat-0dc1cdf91010be057"
	   }
	   public_az_route_table_ids = {
	     "us-east-2a" = "rtb-0642afb4401f1eef1"
	     "us-east-2b" = "rtb-04f511a28a2d5a6a2"
	     "us-east-2c" = "rtb-05f0ee4e831b05697"
	   }
	   public_az_subnet_ids = {
	     "us-east-2a" = "subnet-0dcb9e32f1f02a367"
	     "us-east-2b" = "subnet-0b432a6748ca40638"
	     "us-east-2c" = "subnet-00a9a6636ca722474"
	   }
	   public_az_subnet_cidr_blocks = {
	     "us-east-2a" = "172.16.0.0/21"
	     "us-east-2b" = "172.16.8.0/21"
	     "us-east-2c" = "172.16.16.0/21"
	   }
	*/

	// Run `terraform output` to get the value of an output variable
	privateSubnetIds := terraform.OutputMap(t, terraformOptions, "private_az_subnet_ids")
	privateRouteTableIds := terraform.OutputMap(t, terraformOptions, "private_az_route_table_ids")
	publicNATGateWayIds := terraform.OutputMap(t, terraformOptions, "public_az_ngw_ids")
	publicOnlyNATGateWayIds := terraform.OutputMap(t, terraformOptions, "public_only_az_ngw_ids")
	publicRouteTableIds := terraform.OutputMap(t, terraformOptions, "public_az_route_table_ids")
	publicSubnetIds := terraform.OutputMap(t, terraformOptions, "public_az_subnet_ids")

	expectedAZs := []string{"us-east-2a", "us-east-2b", "us-east-2c"}
	expectedNulls := []string{"<nil>", "<nil>", "<nil>"}
	// Verify we're getting back the outputs we expect
	assert.Equal(t, expectedAZs, getKeys(privateSubnetIds))
	assertValueStartsWith(t, privateSubnetIds, "^subnet-.*")
	assert.Equal(t, expectedAZs, getKeys(privateRouteTableIds))
	assertValueStartsWith(t, privateRouteTableIds, "^rtb-.*")
	assert.Equal(t, expectedAZs, getKeys(publicNATGateWayIds))
	assertValueStartsWith(t, publicNATGateWayIds, "^nat-.*")
	assert.Equal(t, expectedAZs, getKeys(publicOnlyNATGateWayIds))
	assert.Equal(t, expectedNulls, getValues(publicOnlyNATGateWayIds))
	assert.Equal(t, expectedAZs, getKeys(publicRouteTableIds))
	assertValueStartsWith(t, publicRouteTableIds, "^rtb-.*")
	assert.Equal(t, expectedAZs, getKeys(publicSubnetIds))
	assertValueStartsWith(t, publicSubnetIds, "^subnet-.*")

	expectedPublicCidrBlocks := []string{"172.16.0.0/21", "172.16.8.0/21", "172.16.16.0/21"}
	expectedPrivateCidrBlocks := []string{"172.16.128.0/21", "172.16.136.0/21", "172.16.144.0/21"}
	// Run `terraform output` to get the value of an output variable
	publicSubnetCidrBlocks := terraform.OutputMap(t, terraformOptions, "public_az_subnet_cidr_blocks")
	privateSubnetCidrBlocks := terraform.OutputMap(t, terraformOptions, "private_az_subnet_cidr_blocks")
	// Verify output
	assert.Equal(t, expectedAZs, getKeys(publicSubnetCidrBlocks))
	assert.Equal(t, expectedPublicCidrBlocks, getValues(publicSubnetCidrBlocks))
	assert.Equal(t, expectedAZs, getKeys(privateSubnetCidrBlocks))
	assert.Equal(t, expectedPrivateCidrBlocks, getValues(privateSubnetCidrBlocks))
}

func TestExamplesCompleteDisabledModule(t *gotesting.T) {
	// Init phase module download fails when run in parallel
	//t.Parallel()

	terraformOptions := &terraform.Options{
		// The path to where our Terraform code is located
		TerraformDir: "../../examples/complete",
		Upgrade:      true,
		// Variables to pass to our Terraform code using -var-file options
		VarFiles: []string{"fixtures.us-east-2.tfvars"},
		Vars: map[string]interface{}{
			"enabled": "false",
		},
	}

	// At the end of the test, run `terraform destroy` to clean up any resources that were created
	defer terraform.Destroy(t, terraformOptions)

	// This will run `terraform init` and `terraform apply` and fail the test if there are any errors
	terraform.InitAndApply(t, terraformOptions)

	privateNATGateWayIds := terraform.OutputMap(t, terraformOptions, "private_az_ngw_ids")
	privateSubnetIds := terraform.OutputMap(t, terraformOptions, "private_az_subnet_ids")
	privateRouteTableIds := terraform.OutputMap(t, terraformOptions, "private_az_route_table_ids")
	publicNATGateWayIds := terraform.OutputMap(t, terraformOptions, "public_az_ngw_ids")
	publicRouteTableIds := terraform.OutputMap(t, terraformOptions, "public_az_route_table_ids")
	publicSubnetIds := terraform.OutputMap(t, terraformOptions, "public_az_subnet_ids")
	publicSubnetIpv6CidrBlocks := terraform.OutputMap(t, terraformOptions, "public_az_subnet_ipv6_cidr_blocks")

	assert.Empty(t, privateNATGateWayIds)
	assert.Empty(t, privateSubnetIds)
	assert.Empty(t, privateRouteTableIds)
	assert.Empty(t, publicNATGateWayIds)
	assert.Empty(t, publicSubnetIds)
	assert.Empty(t, publicRouteTableIds)
	assert.Empty(t, publicSubnetIpv6CidrBlocks)
}
