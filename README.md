```bash
NAME:
   best-zone-for-spot check-spot - Check best spot capacity in a given region

USAGE:
   best-zone-for-spot check-spot [command options] [arguments...]

OPTIONS:
   --vpc value                        VPC ID to deploy to
   --ami value, --ami-name value      AMI That will be used by the fleet to create instances   (default: "nixos-19.03pre-git-x86_64-hvm-ebs")
   -t value, --instance-type value    list of instance types. Example: ... --instance-type r4.xlarge --instance-type r5.xlarge
   -r value, --region value           AWS region (default: "us-east-1")
   -c value, --target-capacity value  Number of spot instance to bring up to test spot capacity (default: 10)
```

## Build and Install

Note: You need to install nix or you just should install it the default go way

```bash
git clone https://github.com/PsyanticY/best-zone-for-spot
cd best-zone-for-spot
nix-build
nix-env -i path/to/bin/in/nix/store
```