import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.ticker as ticker
import json
import datetime

df = pd.read_csv('data.csv')

# ====== Fairness Analysis Section ======
genesis_author = '0000000000000000000000000000000000000000'
df = df[df['author'] != genesis_author]

# Number of blocks produced by each node
block_counts = df['author'].value_counts().sort_index()

# Load city names from network.json
with open('network.json', 'r') as f:
    network = json.load(f)

nodeId_to_city = {int(node['loc']['id']): node['loc']['name'] for node in network['nodes_view']}

# Map each author to its nodeId and to city name: Author <==> nodeId <==> city name
author_to_nodeid = df.drop_duplicates('author').set_index('author')['nodeId'].to_dict()
labels = [nodeId_to_city.get(author_to_nodeid.get(author, -1), 'Unknown city') for author in block_counts.index]

plt.figure(figsize=(12, 6))
plt.bar(labels, block_counts.values, zorder=20)
plt.xlabel('Node (city name)')
plt.ylabel('Number of produced blocks (#)')
plt.title('Fairness analysis: Blocks produced by each node')
plt.xticks(rotation=90)
plt.gca().yaxis.set_major_locator(ticker.MaxNLocator(integer=True))
plt.rc('axes', axisbelow=True)
plt.grid(axis='y', linestyle='--', alpha=0.7, zorder=0)
plt.tight_layout()
plt.savefig('blocks_per_node.pdf')
plt.show()

print('Fairness statistics):')
print(f"Total nodes (produced at least one block): {block_counts.size}")
print(f"Min blocks: {block_counts.min()}")
print(f"Max blocks: {block_counts.max()}")
print(f"Mean blocks: {block_counts.mean():.2f}")
print(f"Std deviation: {block_counts.std():.2f}")

print(f"Commitments utilization: {((20*8)/(60*3)*100):.2f}%")

# ====== Block Generation Time Analysis Section ======
block_times = df['createdAt'].sort_values().values
block_intervals = block_times[1:] - block_times[:-1]

print('Block Generation Interval Statistics:')
print(f"Min interval: {block_intervals.min()} s")
print(f"Max interval: {block_intervals.max()} s")
print(f"Mean interval: {block_intervals.mean():.2f} s")
print(f"Std deviation: {block_intervals.std():.2f} s")

# Convert createdAt to 8-second intervals
block_time_seconds = 8
start_time = df['createdAt'].min()
end_time = df['createdAt'].max()
bins = range(start_time, end_time + block_time_seconds, block_time_seconds)
df['time_bin'] = pd.cut(df['createdAt'], bins=bins, right=False, labels=bins[:-1])
block_counts_by_bin = df.groupby('time_bin').size()

x_labels = [datetime.datetime.fromtimestamp(int(ts)).strftime('%H:%M:%S') for ts in block_counts_by_bin.index]

plt.figure(figsize=(12, 6))
plt.plot(x_labels, block_counts_by_bin.values, marker='o', zorder=20)
plt.xlabel('Time (8s intervals)')
plt.ylabel('Number of produced blocks (#)')
plt.title('Blocks generated over time (aggregated by 8s)')
plt.rc('axes', axisbelow=True)
plt.grid(axis='y', linestyle='--', alpha=0.7, zorder=0)

tickInterval = max(1, len(x_labels) // 20)

# Reduce the number of x-ticks
plt.xticks(ticks=range(0, len(x_labels), tickInterval), labels=[x_labels[i] for i in range(0, len(x_labels), tickInterval)], rotation=45, ha='right')

plt.tight_layout()
plt.savefig('blocks_over_time.pdf')
plt.show()

# ====== Transaction Throughput Analysis Section ======
start_time = df['createdAt'].min()
end_time = df['createdAt'].max()
total_seconds = end_time - start_time

total_transactions = df['transactionsCount'].sum()

tps = total_transactions / total_seconds if total_seconds > 0 else 0
print(f"Total transactions: {total_transactions}")
print(f"Total time span: {total_seconds} seconds")
print(f"Average throughput: {tps:.2f} transactions/second (TPS)")
# ====== End Transaction Throughput Analysis Section ======
