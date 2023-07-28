KEYPAIR_NAME=ci.test-infra-definitions.test-key
KEYPAIR_PATH=key-pair
ENV=agent-qa

echo "Cleanup"

aws ec2 delete-key-pair --key-name $KEYPAIR_NAME

inv destroy-vm -s ci-integration-testing-$CI_PIPELINE_ID
