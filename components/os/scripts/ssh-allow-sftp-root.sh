# No shebang because this script will be concatenated

# Modify root's key restrictions to allow SFTP connections as root
sed -ie 's/^.* ssh-rsa/restrict,command="internal-sftp" ssh-rsa/' /root/.ssh/authorized_keys