set -e

KEYPAIR_NAME=ci.test-infra-definitions.test-key
KEYPAIR_PATH=key-pair
ENV=agent-qa

echo "Creating ssh key pair"

ssh-keygen -t ed25519 -C "Temporary key pair used for test-infra-definitions integration tests" -f $KEYPAIR_PATH

echo "Import key pair on AWS"

aws ec2 delete-key-pair --key-name $KEYPAIR_NAME
aws ec2 import-key-pair --key-name $KEYPAIR_NAME --public-key-material fileb://$KEYPAIR_PATH.pub


echo "Installing Python dependencies"
pip install -r requirements.txt

echo "Running inv setup"

printf "$ENV\n$KEYPAIR_NAME\nN\n$KEYPAIR_PATH.pub\ntest:ci\n00000000000000000000000000000000\n0000000000000000000000000000000000000000\n" | inv setup

echo "Previous command exited with exit=$?"


echo "Cleanup"

aws ec2 delete-key-pair --key-name $KEYPAIR_NAME
