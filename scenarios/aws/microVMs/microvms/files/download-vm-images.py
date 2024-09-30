#!/usr/bin/env python

import json
import subprocess
import sys
from pathlib import Path


def usage():
    print(f"Usage: {sys.argv[0]} <data-file>")


def main(data_file):
    with open(data_file) as f:
        download_data = json.load(f)

    images_to_download = []

    for image in download_data:
        checksum_file = Path(image["checksum_path"])
        if not check_integrity(checksum_file):
            print(f"Integrity check failed for checksum file: {checksum_file}, downloading image")
            images_to_download.append(image)

    curl_args = ["curl", "--no-progress-meter", "--fail", "--show-error", "--retry", "3", "--parallel"]
    for image in images_to_download:
        curl_args += [image["image_source"], "-o", image["image_path"]]
        curl_args += [image["checksum_source"], "-o", image["checksum_path"]]

    try:
        subprocess.run(curl_args, check=True)
    except subprocess.CalledProcessError:
        print("Failed to download images")
        sys.exit(1)

    failed_integrity = False
    for image in images_to_download:
        if not check_integrity(Path(image["checksum_path"])):
            print(f"Integrity check failed for downloaded image: {image['download_path']}")
            failed_integrity = True

    if failed_integrity:
        print("Some images failed integrity check")
        sys.exit(1)


def check_integrity(checksum_file: Path) -> bool:
    checksum_dir = checksum_file.parent

    try:
        subprocess.run(["sha256sum", "--strict", "--check", checksum_file], cwd=checksum_dir, check=True)
        return True
    except subprocess.CalledProcessError:
        return False


if __name__ == "__main__":
    if len(sys.argv) != 2:
        usage()
        sys.exit(1)

    if sys.argv[1] in ("-h", "--help"):
        usage()
        sys.exit(0)

    main(sys.argv[1])
