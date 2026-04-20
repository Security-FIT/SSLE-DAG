import csv
import json

locations = []
with open('servers.csv', mode ='r') as file:
  csvFile = csv.DictReader(file)
  for row in csvFile:
        locations.append({"id": int(row["id"]), "name": row["name"],
                          "lat": row["latitude"], "lon": row["longitude"]})

with open("servers.json", "w") as output_file:
    output_file.write(json.dumps(locations, indent=4))
