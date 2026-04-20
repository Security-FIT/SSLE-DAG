import pandas as pd
import json

df = pd.read_csv("pings-original.csv")

mapping = {}
duplicates = 0

for index, row in df.iterrows():
    source = int(row["source"])
    dest = int(row["destination"])

    if source not in mapping:
        mapping[source] = {}

    if dest not in mapping[source]:
        mapping[source][dest] = []

    mapping[source][dest].append(float(row["avg"]))

# Sort mapping
mapping = dict(sorted(mapping.items()))
print(f'Mapping size {len(mapping)}')

# Ignore servers with less than 240 connections (something went wrong there)
deletes = [source for source in mapping if len(mapping[source]) < 240]
for key in deletes:
    del mapping[key]

# Manually remove locations where we miss location info
del mapping[292]
del mapping[293]
del mapping[294]
del mapping[295]

target = mapping.keys()
print(f'Target length: {len(target)}')

json_out = []
for source, all_dests in mapping.items():
    mapping[source] = dict(sorted(mapping[source].items()))

    for dest, all_avgs in mapping[source].items():
        mapping[source][dest] = sum(all_avgs) / len(all_avgs)

    print(f'{source} has size of {len(all_dests)}')

    json_out.append({"id": source, "pings": mapping[source]})

# Find intersection for those that have size more than 240
# Missing data will be filled from the other side, otherwise manually
for source, all_dests in mapping.items():
    diff = list(set(target) - set(mapping[source].keys()))

    for one_diff in diff:
        if source in mapping[one_diff]:
            mapping[source][one_diff] = mapping[one_diff][source]
            diff.remove(one_diff)

    print(f"{source}: {diff}")

for source, all_dests in mapping.items():
    print(f'{source} has size of {len(all_dests)}')

    json_out.append({"id": source, "pings": mapping[source]})

with open("pings_original.json", "w") as output_file:
    output_file.write(json.dumps(json_out))
