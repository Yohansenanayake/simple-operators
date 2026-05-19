package controller

import (
	"context"
	"fmt"
	"time"

	computev1 "github.com/Yohansenanayake/simple-operators/api/v1"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func createEc2Instance(ec2Instance *computev1.Ec2Instance) (createdInstanceInfo *computev1.CreatedInstanceInfo, err error) {

	l := log.Log.WithName("createEc2Instance")

	l.Info(" === STARTING EC2 INSTANCE CREATION ===",
		"ami", ec2Instance.Spec.AMIId,
		"instanceType", ec2Instance.Spec.InstanceType,
		"region", ec2Instance.Spec.Region)

	//create the client for EC2 instance creation
	ec2Client := awsClient(ec2Instance.Spec.Region)

	// create the input for RunInstances() API call based on the Ec2Instance spec
	runInput := &ec2.RunInstancesInput{
		ImageId:      aws.String(ec2Instance.Spec.AMIId),                   // aws.String() is used to convert string to *string as required by the AWS SDK
		InstanceType: ec2types.InstanceType(ec2Instance.Spec.InstanceType), // convert string to the specific InstanceType type defined in the AWS SDK
		KeyName:      aws.String(ec2Instance.Spec.KeyPair),
		SubnetId:     aws.String(ec2Instance.Spec.Subnet),
		MinCount:     aws.Int32(1), // we want to create 1 instance
		MaxCount:     aws.Int32(1), // we want to create 1 instance
	}

	l.Info("=== CALLING AWS RunInstances() API ===")
	// run the instances
	result, err := ec2Client.RunInstances(context.TODO(), runInput)
	if err != nil {
		l.Error(err, "Failed to create EC2 instance with RunInstances API")
		return nil, fmt.Errorf("failed to create EC2 instance: %w", err)
	}

	if len(result.Instances) == 0 {
		l.Error(nil, "No instances returned in RunInstancesOutput")
		fmt.Println("No instances returned in RunInstanceesOutput")
		return nil, fmt.Errorf("No instances returned in RunInstancesOutput")
	}

	// Till this point the instance is being created and we have instance ID, private IP, instance type and image id
	// But we may not have the public IP yet and the instance state may still be pending.
	inst := result.Instances[0]
	l.Info("=== EC2 INSTANCE CREATED SUCCESSFULLY ===", "instanceID", aws.ToString(inst.InstanceId))

	// Now we have to wait for the instance to be in running state and then we can get the public IP if allowed
	l.Info("=== WAITING FOR INSTANCE TO BE RUNNING ===")

	runWaiter := ec2.NewInstanceRunningWaiter(ec2Client)
	maxWaitTime := 3 * time.Minute

	err = runWaiter.Wait(context.TODO(), &ec2.DescribeInstancesInput{
		InstanceIds: []string{aws.ToString(inst.InstanceId)},
	}, maxWaitTime)
	if err != nil {
		l.Error(err, "Failed to wait for instance to be running")
		return nil, fmt.Errorf("failed to wait for instance to be running: %w", err)
	}

	// After creating the instance, we waited until is it running
	// Now we can use describe API call and
	// 1. Get the public IP if allowed
	// 2. Get the state of the instance
	// we do this so we can send the instance's state to the status of the CR. for user to see with kubectl get ec2instance
	l.Info("=== CALLING AWS DESCRIBEINSTANCE API TO GET INSTANCE DETAILS ===")
	describeInput := &ec2.DescribeInstancesInput{
		InstanceIds: []string{aws.ToString(inst.InstanceId)},
	}

	describeResult, err := ec2Client.DescribeInstances(context.TODO(), describeInput)
	if err != nil {
		l.Error(err, "Failed to describe EC2 instance after creation")
		return nil, fmt.Errorf("failed to describe EC2 instance after creation: %w", err)
	}

	fmt.Println("Describe result", "public ip", aws.ToString(describeResult.Reservations[0].Instances[0].PublicIpAddress), "state", describeResult.Reservations[0].Instances[0].State.Name)
	// You get "invalid memory address or nil pointer dereference" here if any of the following are true:
	// - result.Instances is nil or has length 0
	// - Any of the pointer fields (e.g., PublicIpAddress, PrivateIpAddress, etc.) are nil

	// To avoid this, always check for nil and length before dereferencing:

	// Wait for a bit to allow instance fields to be populated

	fmt.Printf("Private Ip of the instance: %v", aws.ToString(inst.PrivateIpAddress))
	fmt.Printf("State of the instance: %v", describeResult.Reservations[0].Instances[0].State.Name)
	fmt.Printf("Private DNS of the instance: %v", aws.ToString(inst.PrivateDnsName))
	fmt.Printf("Instance ID of the instance: %v", aws.ToString(inst.InstanceId))
	fmt.Printf("Instance type of the instance: %v", inst.InstanceType)
	fmt.Printf("Image ID of the instance: %v", aws.ToString(inst.ImageId))
	fmt.Printf("Key Name of the instance: %v", aws.ToString(inst.KeyName))

	// block until the instance is running
	// blockUntilInstanceRunning(ctx, ec2Instance.Status.InstanceID, ec2Instance)

	// Get the instance details safely (public IP/DNS might be nil for private subnets)
	instance := describeResult.Reservations[0].Instances[0]
	
	var launchTime *metav1.Time
	if instance.LaunchTime != nil {
		convertedLaunchTime := metav1.NewTime(*instance.LaunchTime)
		launchTime = &convertedLaunchTime
	}

	createdInstanceInfo = &computev1.CreatedInstanceInfo{
		InstanceID: aws.ToString(instance.InstanceId),
		State:      string(instance.State.Name),
		PublicIP:   aws.ToString(instance.PublicIpAddress), // This will be empty string if PublicIpAddress is nil
		PrivateIP:  aws.ToString(instance.PrivateIpAddress),
		PublicDNS:  aws.ToString(instance.PublicDnsName),
		PrivateDNS: aws.ToString(instance.PrivateDnsName),
		LaunchTime: launchTime,
	}

	l.Info("=== EC2 INSTANCE CREATION COMPLETED ===",
		"instanceID", createdInstanceInfo.InstanceID,
		"state", createdInstanceInfo.State,
		"publicIP", createdInstanceInfo.PublicIP,
		"privateIP", createdInstanceInfo.PrivateIP)

	return createdInstanceInfo, nil

}
