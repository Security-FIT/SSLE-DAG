import json

with open('pings_with_self_location.json', 'r') as file:
    data = json.load(file)

# Remove ping data where the key matches the id
for obj in data:
    obj_id = str(obj["id"])  # Convert id to string since ping keys are strings
    if obj_id in obj["pings"]:
        del obj["pings"][obj_id]

with open("loc_pings.json", "w") as output_file:
    output_file.write(json.dumps(data, indent=4))
