import argparse

import yaml
import os
from datetime import datetime
import math
import secrets

from InquirerPy import inquirer
from InquirerPy.validator import PathValidator, NumberValidator

from ruamel.yaml import YAML
import subprocess

SETUP_YAML = "setup.yaml"
DAG_VOLUME_NAME = "dag-ssle-volume"


def check_property_exists(yaml_content, property_name):
    """
    Check if a property exists in the YAML content and exits program if not
    :param yaml_content: The content of the setup.yaml file.
    :param property_name: The name of the property to check.
    """
    if yaml_content is None:
        print(f"Invalid {SETUP_YAML} file: No content found.")
        exit(1)

    if yaml_content[property_name] is None:
        print(f"Invalid {SETUP_YAML} file: No '{property_name}' found.")
        exit(1)


def check_valid_number(yaml_content, property_name):
    check_property_exists(yaml_content, property_name)

    if not isinstance(yaml_content[property_name], int) or int(yaml_content[property_name]) < 0:
        print(f"Invalid value for {property_name} in {SETUP_YAML}")
        exit(1)

def get_docker_compose_content(volume_path, number_of_nodes):
    docker_compose_content = f"""services:
    rabbit-management:
        image: rabbitmq:3-management
        container_name: dag-ssle-rabbitmq
        hostname: dag-ssle-rabbitmq
        ports:
            - '15672:15672'
    dag-ssle-data-processor:
        image: dag-ssle-data-processor:latest
        container_name: dag-ssle-data-processor
        hostname: dag-ssle-data-processor
        ports:
            - '8000:8000'
        volumes:
            - {volume_path}:/volume
    dag-ssle-node-control:
        image: dag-ssle-node-control:latest
        container_name: dag-ssle-node-control
        hostname: dag-ssle-node-control
        ports:
            - '3000:3000'
    dag-ssle-bootstrap-server:
        image: dag-ssle-bootstrap-server:latest
        container_name: dag-ssle-bootstrap-server
        hostname: dag-ssle-bootstrap-server
        volumes:
            - {volume_path}:/volume
    dag-ssle-topology-generator:
        image: dag-ssle-topology-generator:latest
        container_name: dag-ssle-topology-generator
        hostname: dag-ssle-topology-generator
        volumes:
            - {volume_path}:/volume
        command: ['python3', 'topology_generator.py', '-n', '{number_of_nodes}', '-d', 'node_degree.csv']
"""
    for i in range(0, number_of_nodes):
        docker_compose_content += f"""    dag-ssle-node-{i}:
        image: dag-ssle-node-implementation:latest
        container_name: node-{i}
        hostname: node-{i}
        volumes:
            - {volume_path}:/volume
        environment:
            - HOSTNAME_ENV=node-{i}
"""
        
    return docker_compose_content


def is_power_of_two(n):
    """
    Check if a number is a power of two.
    :param n: The number to check.
    :return: True if n is a power of two, False otherwise.
    """
    return (n != 0) and (n & (n - 1)) == 0


def validate_setup(yaml_content):
    """
    Validate the setup.yaml content.
    :param yaml_content: The content of the setup.yaml file.
    """
    check_valid_number(yaml_content, 'nodes')
    check_valid_number(yaml_content, 'block_time_seconds')
    check_valid_number(yaml_content, 'dag_parallel_length')
    check_valid_number(yaml_content, 'epoch_size_rounds')
    check_valid_number(yaml_content, 'tx_per_block')
    check_valid_number(yaml_content, 'min_tx_to_split')
    check_valid_number(yaml_content, 'min_tx_to_merge')
    check_valid_number(yaml_content, 'max_commitments')


def main():
    yaml_loaded = False
    yaml_content = None

    # Parse arguments
    parser = argparse.ArgumentParser()
    parser.add_argument("-n", "--noninteractive", action="store_true", required=False,
                        help=f"Skip user interaction with user but directly read configuration from {SETUP_YAML} "
                             f"instead (requires generated {SETUP_YAML} file)")
    args = parser.parse_args()

    is_noninteractive = args.noninteractive

    # Get the absolute path of the script's directory
    script_dir = os.path.dirname(os.path.abspath(__file__))

    # Change the current working directory to the script's directory
    os.chdir(script_dir)

    # Check and load setup if setup.yaml exists
    file_exists_os = os.path.exists(f"{SETUP_YAML}")
    if file_exists_os:
        with open(SETUP_YAML) as stream:
            try:
                yaml_loaded = True
                yaml_content = yaml.safe_load(stream)
            except yaml.YAMLError:
                yaml_loaded = False
    else:
        yaml_loaded = False

    if yaml_loaded:
        print(f"Loaded {SETUP_YAML} file successfully.")
        validate_setup(yaml_content)

    if yaml_loaded and is_noninteractive:
        if not yaml_content:
            print(f"Invalid {SETUP_YAML} file: No content found.")
            exit(1)

    if is_noninteractive:
        volume_root_path = os.path.abspath(os.getcwd())
        nodes_val = yaml_content['nodes']
        block_time_seconds = yaml_content['block_time_seconds']
        dag_parallel_length = yaml_content['dag_parallel_length']
        epoch_size_rounds = yaml_content['epoch_size_rounds']
        tx_per_block = yaml_content['tx_per_block']
        min_tx_to_split = yaml_content['min_tx_to_split']
        min_tx_to_merge = yaml_content['min_tx_to_merge']
        max_commitments = yaml_content['max_commitments']
        simulate_rounds = yaml_content['simulate_rounds']
        print(f"Configuration set non-interactively.")
    else:
        volume_root_path = inquirer.filepath(
            message="Enter path to create 'volume_{DATETIME}' directory in:",
            validate=PathValidator(is_dir=True, message="Input is not a directory"),
            default=os.path.abspath(os.getcwd()),
            only_directories=True,
        ).execute()

        nodes_val = inquirer.number(
            message="Enter number of nodes <2,240>:",
            min_allowed=2,
            max_allowed=240,
            validate=NumberValidator(float_allowed=False),
            default=yaml_content['nodes'] if yaml_loaded else None,
        ).execute()

        block_time_seconds = inquirer.number(
            message="Enter number required seconds per round <3,60>:",
            min_allowed=3,
            max_allowed=60,
            validate=NumberValidator(float_allowed=False),
            default=yaml_content['block_time_seconds'] if yaml_loaded else None,
        ).execute()

        dag_parallel_length = inquirer.number(
            message="Enter maximum number (power of 2) of branches in DAG blockchain structure <1,64>:",
            min_allowed=1,
            max_allowed=64,
            validate=lambda res: int(res) and is_power_of_two(int(res)),
            default=yaml_content['dag_parallel_length'] if yaml_loaded else None,
        ).execute()

        epoch_size_rounds = inquirer.number(
            message="Enter number of rounds per epoch <1,1000>:",
            min_allowed=1,
            max_allowed=1000,
            validate=NumberValidator(float_allowed=False),
            default=yaml_content['epoch_size_rounds'] if yaml_loaded else None,
        ).execute()

        tx_per_block = inquirer.number(
            message="Enter maximum number of transactions per block <1,10_000>:",
            min_allowed=1,
            max_allowed=10_000,
            validate=NumberValidator(float_allowed=False),
            default=yaml_content['tx_per_block'] if yaml_loaded else None,
        ).execute()

        min_tx_to_split = inquirer.number(
            message="Enter number of transaction required for block to SPLIT <1,10_000>:",
            min_allowed=1,
            max_allowed=10_000,
            validate=NumberValidator(float_allowed=False),
            default=yaml_content['min_tx_to_split'] if yaml_loaded else None,
        ).execute()

        min_tx_to_merge = inquirer.number(
            message="Enter number of transaction required for block to MERGE <1,10_000>:",
            min_allowed=1,
            max_allowed=10_000,
            validate=NumberValidator(float_allowed=False),
            default=yaml_content['min_tx_to_merge'] if yaml_loaded else None,
        ).execute()

        max_commitments = inquirer.number(
            message="Enter maximum number of commitments per node to shuffle per epoch <1,1000>:",
            min_allowed=1,
            max_allowed=1000,
            validate=NumberValidator(float_allowed=False),
            default=yaml_content['max_commitments'] if yaml_loaded else None,
        ).execute()

        simulate_rounds = inquirer.number(
            message="Enter number of rounds to simulate <1,2_000_000>:",
            min_allowed=1,
            max_allowed=2_000_000,
            validate=NumberValidator(float_allowed=False),
            default=yaml_content['simulate_rounds'] if yaml_loaded else None,
        ).execute()

    min_commitments = 1
    pub_key_merkle_size = 512
    pub_key_merkle_tree_depth = int(math.log2(pub_key_merkle_size))
    random_shuffling_key = secrets.token_hex(64)

    # Create volume directory
    volume_dir_path = os.path.join(volume_root_path, f"volume_{datetime.now().strftime('%m-%d-%Y_%H-%M-%S')}")
    os.makedirs(volume_dir_path, exist_ok=True)
    print(f"Created directory: {volume_dir_path}")

    # Create dictionary for yaml config
    yaml_config = {
        'environment': {
            'nodes': int(nodes_val),
            'block_time_seconds': int(block_time_seconds),
            'dag_parallel_length': int(dag_parallel_length),
            'epoch_size_rounds': int(epoch_size_rounds),
            'tx_per_block': int(tx_per_block),
            'min_tx_to_split': int(min_tx_to_split),
            'min_tx_to_merge': int(min_tx_to_merge),
            'random_shuffling_key': random_shuffling_key,
            'pub_key_merkle_size': pub_key_merkle_size,
            'pub_key_merkle_tree_depth': pub_key_merkle_tree_depth,
            'min_commitments': min_commitments,
            'max_commitments': int(max_commitments),
            'simulate_rounds': int(simulate_rounds),
        },
        'bootstrap': {
            'port': 3000
        },
        'rabbitmq': {
            'port': 5672
        },
    }

    # Create yaml object
    yaml_ruamel = YAML()

    # Write the YAML configuration to a file
    with open(os.path.join(volume_dir_path, "config.yaml"), 'w') as yaml_file:
        yaml_file.write("# THIS IS AN AUTOGENERATED FILE. DO NOT EDIT THIS FILE DIRECTLY.\n")
        yaml_ruamel.dump(yaml_config, yaml_file)

    print("Generated config.yaml file in the volume directory.")


    yaml_setup = {
        'nodes': int(nodes_val),
        'block_time_seconds': int(block_time_seconds),
        'dag_parallel_length': int(dag_parallel_length),
        'epoch_size_rounds': int(epoch_size_rounds),
        'tx_per_block': int(tx_per_block),
        'min_tx_to_split': int(min_tx_to_split),
        'min_tx_to_merge': int(min_tx_to_merge),
        'max_commitments': int(max_commitments),
        'simulate_rounds': int(simulate_rounds)
    }

    # Save current setup into setup.yaml for faster interaction next time
    with open(SETUP_YAML, 'w') as yaml_file:
        yaml_file.write("# THIS IS AN AUTOGENERATED FILE. DO NOT EDIT THIS FILE DIRECTLY.\n")
        yaml_ruamel.dump(yaml_setup, yaml_file)

    # Copy zkp-circuit-keys to volume directory 
    subprocess.Popen(f"cp zkp-circuit-keys/pk.bin {os.path.join(volume_dir_path, 'pk.bin')}", shell=True).wait()
    subprocess.Popen(f"cp zkp-circuit-keys/vk.bin {os.path.join(volume_dir_path, 'vk.bin')}", shell=True).wait()

    # Build docker images
    subprocess.Popen('cd topology-generator && docker build --tag dag-ssle-topology-generator .', shell=True).wait()
    subprocess.Popen('cd data-processor && docker build --tag dag-ssle-data-processor .', shell=True).wait()
    subprocess.Popen('cd node-control && docker build --tag dag-ssle-node-control .', shell=True).wait()
    subprocess.Popen('cd bootstrap-server && docker build --tag dag-ssle-bootstrap-server .', shell=True).wait()
    subprocess.Popen('cd node-implementation && docker build --tag dag-ssle-node-implementation .', shell=True).wait()

    with open("docker-compose.yml", 'w') as docker_compose_file:
        docker_compose_file.write(get_docker_compose_content(volume_dir_path, int(nodes_val)))    

    # Remove old docker containers
    subprocess.Popen("docker compose down", shell=True).wait()

    # Start docker compose
    start_docker_compose = subprocess.Popen(f"NODES={nodes_val} VOLUME_PATH={volume_dir_path} docker compose up -d", shell=True)
    start_docker_compose.wait()
    
    if start_docker_compose.returncode != 0:
        print("Error starting Docker containers.")
        exit(1)
    else:
        print("Docker containers started successfully.")


if __name__ == "__main__":
    main()
