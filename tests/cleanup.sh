set -e

echo "Cleanup"
aws ec2 delete-key-pair --key-name ci.test-infra-definitions.test-key-$CI_PIPELINE_ID

inv destroy-vm -s ci-integration-testing-$CI_PIPELINE_ID
