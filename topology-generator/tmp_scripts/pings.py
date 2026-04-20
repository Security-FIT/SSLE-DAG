import json
from collections import defaultdict

with open('pings.json', 'r') as file:
    data = json.load(file)

# Group by id and merge pings
merged_data = defaultdict(lambda: {"id": None, "pings": {}})

for obj in data:
    obj_id = obj["id"]
    if merged_data[obj_id]["id"] is None:
        merged_data[obj_id]["id"] = obj_id
    merged_data[obj_id]["pings"].update(obj["pings"])

# Convert defaultdict back to a list
result = list(merged_data.values())

for obj in result:
    if "292" in obj["pings"].keys():
        del obj["pings"]["292"]
    if "293" in obj["pings"].keys():
        del obj["pings"]["293"]
    if "294" in obj["pings"].keys():
        del obj["pings"]["294"]
    if "295" in obj["pings"].keys():
        del obj["pings"]["295"]

# Output the merged data
# print(json.dumps(result, indent=4))

TARGET = 246
for obj in result:
    if len(obj["pings"].keys()) != TARGET:
        print("Error")

with open("pings_modified.json", "w") as output_file:
    output_file.write(json.dumps(result, indent=4))

