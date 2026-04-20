import json

with open('copy_loc_servers.json', 'r') as file:
    data = json.load(file)

# Remove ping data where the key matches the id
for obj in data:
    obj["id"] = str(obj["id"])

with open("loc_servers.json", "w") as output_file:
    output_file.write(json.dumps(data, indent=4))
