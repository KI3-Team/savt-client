import argparse
import json
import os


# Get or create JSON file
def get_or_create_json(output_path=None):
    json_file = os.path.join(output_path or os.path.dirname(os.path.abspath(__file__)), 'save_client_info.json')

    # If file doesn't exist, create an empty JSON file
    if not os.path.exists(json_file):
        try:
            with open(json_file, 'w') as f:
                json.dump([{"file": "Updates.txt", "link": "/public/sav/Updates.txt"}], f, indent=4)
        except OSError as e:
            print(f"Error creating JSON file: {e}")
            exit(1)

    return json_file


# Read JSON file
def load_json(json_file):
    try:
        with open(json_file, 'r') as f:
            return json.load(f)
    except (OSError, json.JSONDecodeError) as e:
        print(f"Error reading JSON file: {e}")
        return []


# Write JSON file
def save_json(json_file, data):
    try:
        with open(json_file, 'w') as f:
            json.dump(data, f, indent=4)
    except OSError as e:
        print(f"Error writing to JSON file: {e}")
        exit(1)


# Generate description based on arch and os
def get_description(arch, os_type):
    descriptions = {
        ('Windows', 'amd64'): "X86-based PCs with Intel/AMD processors; supports Windows 7 or later",
        ('Windows', 'arm64'): "ARM-based devices, like Surface Pro X; requires Windows 10 or later",
        ('Linux', 'amd64'): "X86_64 architecture",
        ('Linux', 'arm64'): "ARM-based devices",
        ('MacOS', 'amd64'): "Intel-based Macs",
        ('MacOS', 'arm64'): "M1/M2 chips",
    }
    return descriptions.get((os_type, arch), "Unknown build")


# Generate file and link information
def generate_record(version, os ,arch, file_format):
    os_format_name = ""
    if os == 'Windows':
        os_format_name = "windows"
    elif os == 'Linux':
        os_format_name = "linux"
    elif os == 'MacOS':
        os_format_name = "mac"
    else:
        raise ValueError(f"Unsupported os format: {os}")
    file_name = f"savt-client-{version}-{os_format_name}-{arch}.{file_format}"
    link = f"/public/sav/dist/savt-client-{version}/{file_name}"
    return file_name, link


# Update or insert record
def upsert_record(json_file, version, arch, size, sha, file_format, os_type):
    data = load_json(json_file)

    file_name, link = generate_record(version, os_type,arch, file_format)
    description = get_description(arch, os_type)

    # Find record
    existing_record = next(
        (record for record in data if arch in record and record['arch'] == arch and record['os'] == os_type and record['file'] == file_name),
        None
    )

    if existing_record:
        # Update record
        existing_record.update({
            'size': size,
            'sha': sha,
            'link': link,
            'description': description
        })
        print(f"Record updated: {file_name}")
    else:
        # Insert new record
        new_record = {
            "file": file_name,
            "link": link,
            "os": os_type,
            "arch": arch,
            "size": size,
            "sha": sha,
            "description": description
        }
        data.append(new_record)
        print(f"Record inserted: {file_name}")

    save_json(json_file, data)


# Clear JSON file
def clear_json(json_file):
    save_json(json_file, [])
    print(f"JSON file cleared: {json_file}")


# List all records
def list_records(json_file):
    data = load_json(json_file)
    if data:
        print("Existing records:")
        for record in data:
            print(json.dumps(record, indent=4))
    else:
        print("No records found.")


# Main function
def main():
    parser = argparse.ArgumentParser(description="Manage save_client_info.json records.")
    parser.add_argument('--action', choices=['upsert', 'clear', 'list','adjust'], default='list', help="Action to perform.")

    # Parameters needed for upsert
    parser.add_argument('--version', help="Version of the file.")
    parser.add_argument('--arch', choices=['arm64', 'amd64'], required=False,help="Architecture of the file.")
    parser.add_argument('--size', help="Size of the file.")
    parser.add_argument('--sha', help="SHA of the file.")
    parser.add_argument('--file_format', help="Format of the file (e.g., exe, dmg, tar.gz).")
    parser.add_argument('--os', choices=['Windows', 'MacOS', 'Linux'],required=False, help="Operating system of the file.")

    # output_path parameter
    parser.add_argument('--output_path', help="Path where save_client_info.json is located.", required=True)

    args = parser.parse_args()

    # Get JSON file path
    json_file = get_or_create_json(args.output_path)

    if args.action == 'upsert':
        if not all([args.version, args.arch, args.size, args.sha, args.file_format, args.os]):
            print("Missing required parameters for upsert action.")
            exit(1)
        upsert_record(json_file, args.version, args.arch, args.size, args.sha, args.file_format, args.os)
    elif args.action == 'adjust':
        # Parse JSON file content and move array first element to last
        data = load_json(json_file)
        if data and isinstance(data, list):
            data.append(data.pop(0))
            save_json(json_file, data)
    elif args.action == 'clear':
        clear_json(json_file)
    elif args.action == 'list':
        list_records(json_file)


if __name__ == '__main__':
    main()