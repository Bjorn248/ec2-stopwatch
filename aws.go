package main

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/ec2"
)

const (
	north_virginia   string = "us-east-1"
	oregon           string = "us-west-2"
	north_california string = "us-west-1"
	ireland          string = "eu-west-1"
	frankfurt        string = "eu-central-1"
	singapore        string = "ap-southeast-1"
	sydney           string = "ap-southeast-2"
	tokyo            string = "ap-northeast-1"
	sao_paolo        string = "sa-east-1"
)

var regions map[string]int = map[string]int{
	north_virginia:   1,
	oregon:           1,
	north_california: 1,
	ireland:          1,
	frankfurt:        1,
	singapore:        1,
	sydney:           1,
	tokyo:            1,
	sao_paolo:        1,
}

var invalidRegionError error = errors.New("Region string not valid")

/*
Starts an ec2 instance. To be used in combination with
the cron library

Input Params
Type - Description - Name

String - AWS Access Key ID - AccessKeyID
String - AWS Secret Key ID - SecretKeyID
String - AWS EC2 Instance ID - InstanceID
String - AWS Region - Region

Returns
TODO Determine return values
*/
func startInstance(AccessKeyID string, SecretKeyID string, InstanceID string, Region string) (string, error) {
	// Ensure that region is valid
	_, ok := regions[Region]
	if ok == false {
		return "", invalidRegionError
	}

	// Initialize AWS Credentials
	creds := credentials.NewStaticCredentials(AccessKeyID, SecretKeyID, "")

	// Initialize ec2 service
	svc := ec2.New(&aws.Config{Credentials: creds, Region: aws.String(Region)})

	params := &ec2.StartInstancesInput{
		InstanceIds: []*string{
			aws.String(InstanceID),
		},
		DryRun: aws.Bool(false),
	}
	resp, err := svc.StartInstances(params)
	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return "", err
	}
	fmt.Println(resp)

	return "", nil
}
