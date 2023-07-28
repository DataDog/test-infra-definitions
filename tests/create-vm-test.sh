KEYPAIR_NAME=ci.test-infra-definitions.test-key
KEYPAIR_PATH=key-pair
ENV=agent-qa

set -e

echo "Creating ssh key pair"

ssh-keygen -t ed25519 -C "Temporary key pair used for test-infra-definitions integration tests" -f key-pair

echo "Import key pair on AWS"

aws ec2 delete-key-pair --key-name ci.test-infra-definitions.test-key-$CI_PIPELINE_ID
aws ec2 import-key-pair --key-name ci.test-infra-definitions.test-key-$CI_PIPELINE_ID --public-key-material fileb://key-pair.pub

echo "Running inv setup"
printf "$ENV\nci.test-infra-definitions.test-key-$CI_PIPELINE_ID\nN\nkey-pair.pub\ntest-ci\n00000000000000000000000000000000\n0000000000000000000000000000000000000000\n" | inv setup
echo "Successfuly ran inv setup"

echo "Running inv create-vm"
export PULUMI_CONFIG_PASSPHRASE=$(head /dev/urandom | tr -dc A-Za-z0-9 | head -c10)
inv create-vm -s ci-integration-testing-$CI_PIPELINE_ID --private-key-path $KEYPAIR_PATH
