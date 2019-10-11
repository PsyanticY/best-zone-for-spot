package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/urfave/cli"
)

//// Fetch nixos latest image

func fetchImageID(ec2_client *ec2.EC2, amiName string) (string, error) {
	params := &ec2.DescribeImagesInput{
		Filters: []*ec2.Filter{
			{
				Name: aws.String("name"),
				Values: []*string{
					aws.String(amiName),
				},
			},
		},
	}

	result, err := ec2_client.DescribeImages(params)
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}
		return "", err
	}
	image := ""
	fmt.Println("[+] Getting latest NixOS ami to use in the launch template")
	fmt.Printf("[+] AMI name: %q\n", amiName)
	for _, img := range result.Images {
		fmt.Printf("[+] Description: %q\n", *img.Description)
		fmt.Printf("[+] ImageId: %q\n", *img.ImageId)
		image = *img.ImageId
	}
	return image, nil

}
func buildLaunchTemplateData(image string) (*ec2.RequestLaunchTemplateData, error) {
	opts := &ec2.RequestLaunchTemplateData{
		UserData: aws.String("test test"),
	}
	opts.ImageId = aws.String(image)
	// the choice is arbitrary since we won't be creating this.
	opts.InstanceType = aws.String("m4.large")

	return opts, nil
}

func main() {
	app := cli.NewApp()
	app.Compiled = time.Now()
	app.Authors = []cli.Author{
		cli.Author{
			Name:  "PsyanticY",
			Email: "iuns@outlook.fr",
		},
	}
	app.Copyright = "MIT PsyanticY (2019)"
	app.UsageText = "Use ec2 fleet capacity optimazed to provide the best zone to deploy spot instances"
	app.Name = "Find you the best spot capacity zone"
	app.Usage = "Provide insight on the best zone to spin up spot instances"
	app.Version = "0.0.1"
	myFlags := []cli.Flag{
		cli.StringFlag{
			Required: true,
			Name:     "vpc",
			Value:    "",
			Usage:    "VPC ID to deploy to",
		},
		cli.StringFlag{
			Name:  "ami, ami-name",
			Value: "nixos-19.03pre-git-x86_64-hvm-ebs",
			Usage: "AMI That will be used by the fleet to create instances	",
		},
		cli.StringSliceFlag{
			Name:     "t, instance-type",
			Required: true,
			Usage:    "list of instance types. Example: ... --instance-type r4.xlarge --instance-type r5.xlarge",
		},
		cli.StringFlag{
			Name:  "r, region",
			Value: "us-east-1",
			Usage: "AWS region",
		},
		cli.Int64Flag{
			Name:  "c, target-capacity",
			Value: 10,
			Usage: "Number of spot instance to bring up to test spot capacity",
		},
	}
	app.EnableBashCompletion = true
	// we create our commands
	app.Commands = []cli.Command{
		{
			Name:    "check-spot",
			Aliases: []string{"cs"},
			Usage:   "Check best spot capacity in a given region",
			Flags:   myFlags,
			Action: func(c *cli.Context) error {
				instanceType := c.StringSlice("instance-type")
				region := c.String("region")
				vpc := c.String("vpc")
				amiName := c.String("ami-name")
				targetCapacity := c.Int64("target-capacity")

				fmt.Println("--------------------------------------------------------------------")
				fmt.Println("--------------------------------------------------------------------")

				client, err := session.NewSession(&aws.Config{
					Region: aws.String(region)},
				)
				// Create a sts service client.
				ec2_client := ec2.New(client)

				// get vpc subnets
				vpcInput := &ec2.DescribeSubnetsInput{
					Filters: []*ec2.Filter{
						{
							Name: aws.String("vpc-id"),
							Values: []*string{
								aws.String(vpc),
							},
						},
					},
				}

				vpcResult, vpcerr := ec2_client.DescribeSubnets(vpcInput)
				if err != nil {
					if aerr, ok := err.(awserr.Error); ok {
						switch aerr.Code() {
						default:
							fmt.Println(aerr.Error())
						}
					} else {
						// Print the error, cast err to awserr.Error to get the Code and
						// Message from an error.
						fmt.Println(err.Error())
					}
					return vpcerr
				}
				mapZoneToSubnet := make(map[string]string)
				mapZoneToIPAddressCount := make(map[string]int64)
				for _, subnet := range vpcResult.Subnets {
					if _, ok := mapZoneToSubnet[*subnet.AvailabilityZone]; ok {
						if mapZoneToIPAddressCount[*subnet.AvailabilityZone] < *subnet.AvailableIpAddressCount {
							mapZoneToSubnet[*subnet.AvailabilityZone] = *subnet.SubnetId
							mapZoneToIPAddressCount[*subnet.AvailabilityZone] = *subnet.AvailableIpAddressCount
						}
					} else {
						mapZoneToSubnet[*subnet.AvailabilityZone] = *subnet.SubnetId
						mapZoneToIPAddressCount[*subnet.AvailabilityZone] = *subnet.AvailableIpAddressCount
					}
				}

				// getting valid subnet list
				fmt.Printf("[+]\n")
				fmt.Printf("[+] Gathering subnets that will be used in the request from: %q\n", vpc)
				fmt.Printf("[+]\n")
				var subnetList []string
				for j, i := range mapZoneToSubnet {
					subnetList = append(subnetList, i)
					fmt.Printf("[+] subnet:%q, in zone:%q\n", i, j)
				}
				fmt.Printf("[+]\n")
				// get nixos latest image:
				image, err := fetchImageID(ec2_client, amiName)
				// work on ec2 launch template:
				fmt.Printf("[+]\n")
				fmt.Printf("[+] Working on the launch template\n")
				launchTemplateData, err := buildLaunchTemplateData(image)
				if err != nil {
					return err
				}

				templateName := "lt-for-ec2-fleet"
				var ltID string
				launchTemplateOpts := &ec2.CreateLaunchTemplateInput{
					LaunchTemplateName: aws.String(templateName),
					LaunchTemplateData: launchTemplateData,
					VersionDescription: aws.String("lt used to test spot availability"),
				}

				ltResult, err := ec2_client.CreateLaunchTemplate(launchTemplateOpts)
				if err != nil {
					if aerr, ok := err.(awserr.Error); ok {
						switch aerr.Code() {
						case "InvalidLaunchTemplateName.AlreadyExistsException":
							// get temlpateid

							existingTemplateInputs := &ec2.DescribeLaunchTemplatesInput{
								LaunchTemplateNames: []*string{
									aws.String(templateName),
								},
							}

							existingTemplateOutputs, err := ec2_client.DescribeLaunchTemplates(existingTemplateInputs)
							ltID = *existingTemplateOutputs.LaunchTemplates[0].LaunchTemplateId
							fmt.Printf("[+] Note: Launch template arleady exits, will be using it\n")
							if err != nil {
								return err
							}
						default:
							return err
						}
					}
				} else {
					launchTemplate := ltResult.LaunchTemplate
					ltID = *launchTemplate.LaunchTemplateId
				}

				fmt.Printf("[+] Launch Template created: %q\n", ltID)

				// ec2 fleet stuff
				fmt.Printf("[+]\n")
				fmt.Printf("[+] Building ec2 fleet request\n")

				overridesCount := len(instanceType) * len(subnetList) // number of subnets * instance types
				FleetLaunchTemplateSpecificationRequest := &ec2.FleetLaunchTemplateSpecificationRequest{
					LaunchTemplateId: aws.String(ltID),
					Version:          aws.String("1"),
				}

				FleetLaunchTemplateOverridesRequests := make([]*ec2.FleetLaunchTemplateOverridesRequest, overridesCount)
				k := 0
				j := 0
				for i := range FleetLaunchTemplateOverridesRequests {
					FleetLaunchTemplateOverridesRequests[i] = &ec2.FleetLaunchTemplateOverridesRequest{
						SubnetId:     aws.String(subnetList[k]),
						InstanceType: aws.String(instanceType[j]),
					}
					k++
					if k == len(subnetList) {
						k = 0
						j++
					}
				}

				fleetLaunchTemplateConfigRequests := make([]*ec2.FleetLaunchTemplateConfigRequest, 1)
				for i := range fleetLaunchTemplateConfigRequests {

					fleetLaunchTemplateConfigRequests[i] = &ec2.FleetLaunchTemplateConfigRequest{
						LaunchTemplateSpecification: FleetLaunchTemplateSpecificationRequest,
						Overrides:                   FleetLaunchTemplateOverridesRequests,
					}
				}

				OnDemandOptions := &ec2.OnDemandOptionsRequest{
					AllocationStrategy:     aws.String("lowestPrice"),
					MinTargetCapacity:      aws.Int64(1),
					SingleAvailabilityZone: aws.Bool(false),
					SingleInstanceType:     aws.Bool(false),
				}
				SpotOptionsRequest := &ec2.SpotOptionsRequest{
					AllocationStrategy:           aws.String("capacityOptimized"),
					InstanceInterruptionBehavior: aws.String("terminate"),
					// valid only if allocationb strategy is lowestPrice
					// InstancePoolsToUseCount:      aws.Int64(3),
					MaxTotalPrice:          aws.String("999"),
					MinTargetCapacity:      aws.Int64(1),
					SingleAvailabilityZone: aws.Bool(false),
					SingleInstanceType:     aws.Bool(false),
				}
				TargetCapacitySpecification := &ec2.TargetCapacitySpecificationRequest{
					DefaultTargetCapacityType: aws.String("spot"),
					OnDemandTargetCapacity:    aws.Int64(0),
					SpotTargetCapacity:        aws.Int64(1),
					TotalTargetCapacity:       aws.Int64(targetCapacity),
				}
				fleetInputs := &ec2.CreateFleetInput{
					ExcessCapacityTerminationPolicy:  aws.String("termination"),
					LaunchTemplateConfigs:            fleetLaunchTemplateConfigRequests,
					OnDemandOptions:                  OnDemandOptions,
					ReplaceUnhealthyInstances:        aws.Bool(false),
					SpotOptions:                      SpotOptionsRequest,
					TargetCapacitySpecification:      TargetCapacitySpecification,
					TerminateInstancesWithExpiration: aws.Bool(true),
					Type:                             aws.String("request"),
				}

				// doing the actual request
				fmt.Printf("[+] EC2 Fleet request\n")
				fleetRequest, fleetErr := ec2_client.CreateFleet(fleetInputs)
				if fleetErr != nil {
					return fleetErr
				}
				fleetID := *fleetRequest.FleetId
				time.Sleep(15 * time.Second)
				fmt.Printf("[+] Wating for ec2 fleet request to be fulfilled.")
				retries := 0
				// waiting for the spot fleet to get fulfulled
				describeFleetsInputs := &ec2.DescribeFleetsInput{
					FleetIds: []*string{
						aws.String(fleetID),
					},
				}
				fleetStatus, err := ec2_client.DescribeFleets(describeFleetsInputs)
				if err != nil {
					return err
				}
				for true {
					retries++

					fleetStatus, err = ec2_client.DescribeFleets(describeFleetsInputs)
					if err != nil {
						return fleetErr
					}
					fmt.Printf(".")
					if *fleetStatus.Fleets[0].ActivityStatus == "fulfilled" {
						fmt.Printf("fulfilled\n")
						break
					} else if *fleetStatus.Fleets[0].ActivityStatus == "error" {
						panic("ec2 fleet activity status is error; check your config")
					} else {
						time.Sleep(2 * time.Second)
						if retries == 30 {
							break
							// for some reason after break the progream exit entireley
						}
					}
				}
				desctibeFleetInstancesInputs := &ec2.DescribeFleetInstancesInput{
					FleetId: aws.String(fleetID),
				}
				fleetInstances, errDescribe := ec2_client.DescribeFleetInstances(desctibeFleetInstancesInputs)

				if errDescribe != nil {
					return errDescribe
				}

				fmt.Printf("[+] Out of the desired capacity of %v, EC2 fleet provisioned %v instances\n",
					targetCapacity, len(fleetInstances.ActiveInstances))
				fmt.Printf("[+]\n")
				for _, instance := range fleetInstances.ActiveInstances {

					instanceInputs := &ec2.DescribeInstanceStatusInput{
						InstanceIds: []*string{
							aws.String(*instance.InstanceId),
						},
					}
					instanceStatus, err := ec2_client.DescribeInstanceStatus(instanceInputs)
					if err != nil {
						return err
					}
					fmt.Printf("[+] InstanceID:%q, Zone:%q, InstanceType:%q\n",
						*instance.InstanceId, *instanceStatus.InstanceStatuses[0].AvailabilityZone, *instance.InstanceType)
				}
				fmt.Printf("[-]\n")
				// // Starting Clean up:
				fmt.Printf("[-] Starting clean up...\n")
				// destroy fleet:
				fmt.Printf("[-] Deleting ec2 fleet: %v\n", fleetID)
				DeleteFleetsInput := &ec2.DeleteFleetsInput{
					FleetIds: []*string{
						aws.String(fleetID),
					},
					TerminateInstances: aws.Bool(true),
				}

				resp, err := ec2_client.DeleteFleets(DeleteFleetsInput)
				if err != nil {
					return err
				}
				fmt.Println(resp)
				fmt.Printf("[-] ec2 fleet deleted: %v\n", fleetID)
				//Destroy ec2 launch template
				// panic(image)

				fmt.Printf("[-] Launch Template to delete: %v\n", ltID)
				_, errr := ec2_client.DeleteLaunchTemplate(&ec2.DeleteLaunchTemplateInput{
					LaunchTemplateId: aws.String(ltID),
				})

				if errr != nil {
					return errr
				}

				fmt.Printf("[-] Launch Template deleted: %v\n", ltID)

				fmt.Println("--------------------------------------------------------------------")
				fmt.Println("--------------------------------------------------------------------")
				return nil
			},
		},
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
