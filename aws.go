package main

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/route53"
	"time"
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
var unknownIPError error = errors.New("Unable to retrieve IP after 5 minutes")

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
TODO Determine proper return values
*/
func startInstance(AccessKeyID string, SecretKeyID string, InstanceID string, Region string, ddns DDNS) (*ec2.StartInstancesOutput, error) {
	// Ensure that region is valid
	_, ok := regions[Region]
	if ok == false {
		return &ec2.StartInstancesOutput{}, invalidRegionError
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
		return &ec2.StartInstancesOutput{}, err
	}

	if ddns.Enabled == "true" {
		go dynamicDNS(AccessKeyID, SecretKeyID, InstanceID, Region, ddns.Domain, ddns.HostedZoneID)

	}

	return resp, nil
}

func stopInstance(AccessKeyID string, SecretKeyID string, InstanceID string, Region string) (*ec2.StopInstancesOutput, error) {
	// Ensure that region is valid
	_, ok := regions[Region]
	if ok == false {
		return &ec2.StopInstancesOutput{}, invalidRegionError
	}

	// Initialize AWS Credentials
	creds := credentials.NewStaticCredentials(AccessKeyID, SecretKeyID, "")

	// Initialize ec2 service
	svc := ec2.New(&aws.Config{Credentials: creds, Region: aws.String(Region)})

	params := &ec2.StopInstancesInput{
		InstanceIds: []*string{
			aws.String(InstanceID),
		},
		DryRun: aws.Bool(false),
	}
	resp, err := svc.StopInstances(params)
	if err != nil {
		// Print the error, cast err to awserr.Error to get the Code and
		// Message from an error.
		fmt.Println(err.Error())
		return &ec2.StopInstancesOutput{}, err
	}

	return resp, nil
}

func getPublicIPOfInstance(AccessKeyID string, SecretKeyID string, InstanceID string, Region string) (*ec2.DescribeInstancesOutput, error) {
	// Ensure that region is valid
	_, ok := regions[Region]
	if ok == false {
		return &ec2.DescribeInstancesOutput{}, invalidRegionError
	}

	// Initialize AWS Credentials
	creds := credentials.NewStaticCredentials(AccessKeyID, SecretKeyID, "")

	// Initialize ec2 service
	svc := ec2.New(&aws.Config{Credentials: creds, Region: aws.String(Region)})

	params := &ec2.DescribeInstancesInput{
		InstanceIds: []*string{
			aws.String(InstanceID),
		},
		DryRun: aws.Bool(false),
	}

	for i := 0; i < 10; i++ {
		resp, err := svc.DescribeInstances(params)
		if err != nil {
			fmt.Println(err.Error())
			return &ec2.DescribeInstancesOutput{}, err
		}
		if resp.Reservations[0].Instances[0].PublicIpAddress == nil {
			fmt.Println("no public ip")
			time.Sleep(30 * time.Second)
			continue
		}
		return resp, nil
	}
	return &ec2.DescribeInstancesOutput{}, unknownIPError

}

// This function sets the A record of a domain to the IP of a running instance
func dynamicDNS(AccessKeyID string, SecretKeyID string, InstanceID string, Region string, Domain string, HostedZoneID string) error {
	// Get IP of freshly booted instance
	publicIpResponse, ipErr := getPublicIPOfInstance(AccessKeyID, SecretKeyID, InstanceID, Region)
	if ipErr != nil {
		fmt.Println(ipErr)
		return ipErr
	}

	IP := publicIpResponse.Reservations[0].Instances[0].PublicIpAddress
	creds := credentials.NewStaticCredentials(AccessKeyID, SecretKeyID, "")

	// Initialize Route53 Service
	svc := route53.New(&aws.Config{Credentials: creds})

	params := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name: aws.String(Domain),
						Type: aws.String(route53.RRTypeA),
						ResourceRecords: []*route53.ResourceRecord{
							{
								Value: IP,
							},
						},
						TTL: aws.Int64(300),
					},
				},
			},
			Comment: aws.String("Updating Domain for EC2 Instance via Stopwatch"),
		},
		HostedZoneId: aws.String(HostedZoneID),
	}
	_, err := svc.ChangeResourceRecordSets(params)

	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	return nil
}
