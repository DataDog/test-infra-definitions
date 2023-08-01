KEYPAIR_NAME=ci.test-infra-definitions.test-key
KEYPAIR_PATH=key-pair
ENV=agent-qa

echo "Running inv setup"
printf "$ENV\n$E2E_KEY_PAIR_NAME\nN\n$E2E_PUBLIC_KEY_PATH\ntest-ci\n00000000000000000000000000000000\n0000000000000000000000000000000000000000\n" | inv setup
setup_exit_code=$?

echo "Running inv create-vm"
export PULUMI_CONFIG_PASSPHRASE=$(head /dev/urandom | tr -dc A-Za-z0-9 | head -c10)
inv create-vm -s ci-integration-testing-$CI_PIPELINE_ID
create_exit_code=$?

echo "Cleanup"
inv destroy-vm -s ci-integration-testing-$CI_PIPELINE_ID --yes
destroy_exit_code=$?

echo "Test results"

failed=false
if [[ $setup_exit_code -eq 0 ]]; then
    echo "invoke setup worked successfuly"
else
    echo "invoke setup failed"
    failed=true
fi

if [[ $destroy_exit_code -eq 0 ]]; then
    echo "invoke create-vm worked successfuly"
else
    echo "invoke create-vm failed"
    failed=true
fi

if [[ $destroy_exit_code -eq 0 ]]; then
    echo "invoke destroy-vm worked successfuly"
else
    echo "invoke destroy-vm failed"
    failed=true
fi

if [[ $failed == true ]]; then
    exit 1
fi


