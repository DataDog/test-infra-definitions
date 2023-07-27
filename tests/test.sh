KEYPAIR_NAME=ci.test-infra-definitions.test-key
KEYPAIR_PATH=key-pair
ENV=agent-qa

echo "Creating ssh key pair"

ssh-keygen -t ed25519 -C "Temporary key pair used for test-infra-definitions integration tests" -f $KEYPAYR_PATH

echo "Import key pair on AWS"
aws ec2 import-key-air --key-name $KEYPAIR_NAME --public-key-material fileb:://$KEYPAIR_PATH -C "Temporary ssh key used for invoke task integration testing in test-infra-definitions CI"


echo "Running inv setup"



printf "$ENV\n$KEYPAIR_NAME\nN\n$KEYPAIR_PATH.pub\ntest:ci\n000\n\n" | inv setup

echo "Previous command exited with exit=$?"


echo "Cleanup"

aws ec2 delete-key-pair --key-name $KEYPAIR_NAME
