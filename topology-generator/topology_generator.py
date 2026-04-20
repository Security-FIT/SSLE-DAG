import argparse
import random
import pandas as pd
import networkx as nx
from tqdm import tqdm
from scipy import stats
import os
import sys
import copy
import math
import json
import statistics

STORE_ENV_NAME = "STORE_ENV_NAME"

def main():
    # Parse arguments
    parser = argparse.ArgumentParser()
    parser.add_argument("-n", "--nodes", type=int, required=True, help="Node count")
    parser.add_argument("-d", "--node_degree", type=str, required=True, help="Node degree data filepath")
    parser.add_argument("-o", "--output", type=str, required=False, help="Output file (e.g., out.json)")

    args = parser.parse_args()

    # Check if output argument is disabled in deploy environment
    if STORE_ENV_NAME in os.environ and args.output is not None:
        print(f"--output argument is not allowed in deploy environment (using ENV variable {STORE_ENV_NAME})", file=sys.stderr)
        exit(1)

    if STORE_ENV_NAME in os.environ:
        path_to_save = f"{os.environ[STORE_ENV_NAME]}/network.json"
    elif args.output is not None:
        path_to_save = args.output
    else:
        path_to_save = "network.json"

    print(path_to_save)

    # Check if file on specified path already exists
    if (os.path.exists(path_to_save)):
        print(f"File {path_to_save} already exists. Aborting...", file=sys.stderr)
        exit(1)

    # Check if path for node_degree exists
    if not os.path.exists(args.node_degree):
        print("Specified path to data file does not exists", file=sys.stderr)
        exit(1)
    
    # ============ 1. Initialization ============

    # Load node_degree data and create discrete random generator based on probabilities
    df = pd.read_csv(args.node_degree)
    node_degree_gen = stats.rv_discrete(
        name="node_degree", seed=12, values=(list(df["node-degree"]), list(df["probabilities"]))
    )

    # Create initial state of the network
    total_node_count = args.nodes
    g = nx.Graph()
    total_edges = 0
    edge_drop_counter = 0

    # ============================================================================
    # Configuration variables, can be adjusted for particular use cases

    # Number of nodes to test their edge connection score
    END_NODES_TEST = math.ceil(total_node_count - 1)

    # Number of nodes from which an edge score is computed (average hop count)
    PATH_NODES_TEST = math.ceil(total_node_count / 2)

    # Add restrictions for maximum differencies to test
    MAX_DIFF_EDGES_TRY = 30
    MAX_EDGES_COUNT_TRY = 1
    # ============================================================================

    if END_NODES_TEST >= total_node_count:
        print("Nodes to test must smaller than all nodes in output topology")
        exit(1)

    # Init struct to keep record of total and free numbers of edges per node
    node_peers = {}

    # ============ 2. Nodes Loop ============

    # Add nodes into the graph structure and set their expected number of edges (number of edges might drop further)
    for i in range(0, total_node_count):
        # Add node to graph structure
        g.add_node(i)

        # Set each node number of expected peers
        edges = node_degree_gen.rvs()

        # Modify number of peers for small network
        if edges > total_node_count - 1:
            edges = math.ceil(total_node_count - 1 / 2)

        total_edges += edges
        node_peers[i] = {"total": edges, "free": edges}

    # ============ 3. Edges loop ============

    # Algorithm to add edges
    for i in tqdm(range(0, total_node_count)):
        edges = 0

        edges_count_try = 0
        while edges_count_try < MAX_EDGES_COUNT_TRY:

            # If it was not possible to find some node to connect, cut off remaining edges
            # This approach, however do not have to necesseraly lower its final amount of peers
            # as this node can still receive connection from other node
            if edges_count_try + 1 < MAX_EDGES_COUNT_TRY or MAX_EDGES_COUNT_TRY == 1:
                edges = node_peers[i]["free"]
            else:
                edges = 0
                if MAX_DIFF_EDGES_TRY != 1:
                    edge_drop_counter += node_peers[i]["free"]

            for _ in range(0, edges):
                edge_added = 0

                while edge_added < MAX_DIFF_EDGES_TRY:
                    start_node = i
                    # Sample a set of end nodes (possible node to connect with start node)
                    end_nodes = random.sample(range(0, total_node_count), END_NODES_TEST)
                    scores = {}

                    # Create two sets of peers to perform a pair test of how this connection
                    # between START_NODE and END_NODE affects path between nodes T1 and T2
                    # These sets can contain same nodes but they have to differ at each pair position
                    # If there is any duplication it will be resolved further
                    test_nodes1 = [random.randrange(total_node_count) for _ in range(PATH_NODES_TEST)]
                    test_nodes2 = [random.randrange(total_node_count) for _ in range(PATH_NODES_TEST)]

                    # Replace start node if found in end nodes set
                    while start_node in end_nodes:
                        end_nodes = random.sample(range(0, total_node_count), END_NODES_TEST)

                    # Fill initial score
                    for n in end_nodes:
                        scores[n] = 0

                    # Iterate all end nodes
                    for n in range(0, len(end_nodes)):
                        # For each END_NODE perform multiple tests between T1 and T2 and calculate average which determines
                        # score for the particular END_NODE
                        end_node = end_nodes[n]
                        newG = copy.deepcopy(g)
                        newG.add_edge(start_node, end_node)

                        hops = []
                        for t in range(0, len(test_nodes1)):
                            # If found duplicate at pair position, replace duplicate node in test nodes sets
                            while test_nodes1[t] == test_nodes2[t]:
                                test_nodes2[t] = random.randint(0, total_node_count - 1)

                            # Compute path and assign number of hops if path was found
                            try:
                                path = list(
                                    nx.shortest_path(
                                        newG, source=test_nodes1[t], target=test_nodes2[t], method="dijkstra"
                                    )
                                )
                                hops.append(len(path))
                            except:
                                hops.append(0)

                        # Calculate score for the END_NODE
                        avg_hop_count = sum(hops) / len(hops)
                        scores[end_nodes[n]] = avg_hop_count

                    # Sort scores for end nodes
                    sorted_score_keys = list(dict(sorted(scores.items(), key=lambda item: item[1])).keys())

                    while True:
                        if len(sorted_score_keys) == 0:
                            edge_added += 1
                            if MAX_EDGES_COUNT_TRY == 1 and edge_added == MAX_DIFF_EDGES_TRY:
                                edge_drop_counter += 1
                            break

                        # To avoid selecting only the best nodes and introduce more entrophy
                        # randomly select END_NODE from the end nodes half with the better performance
                        new_end_node = sorted_score_keys[
                            random.randint(
                                math.ceil((len(sorted_score_keys) - 1) / 2),
                                0 if len(sorted_score_keys) == 0 else len(sorted_score_keys) - 1,
                            )
                        ]

                        # Create new edge connection if there is space for selected END_NODE to add this connection
                        if node_peers[new_end_node]["free"] > 0:
                            g.add_edge(start_node, new_end_node)
                            node_peers[start_node]["free"] -= 1
                            node_peers[new_end_node]["free"] -= 1
                            edge_added = 2 * MAX_DIFF_EDGES_TRY
                            break
                        else:
                            sorted_score_keys.remove(new_end_node)

            # Keep track of missed edges with any end_node and check further in order to avoid loop on a single node
            if edge_added != MAX_DIFF_EDGES_TRY or edges == 0:
                edges_count_try = 2 * MAX_EDGES_COUNT_TRY
            else:
                edges_count_try += 1

    # ============ End of algorithm ============

    # Store number of peers for stats in output
    num_of_peers = []
    for i in range(0, len(list(nx.nodes(g)))):
        num_of_peers.append(len(list(nx.neighbors(g, i))))

    # Load servers and ping info
    with open("loc_pings.json", "r") as pings_file, open("loc_servers.json", "r") as servers_file:
        pings = json.load(pings_file)
        servers = json.load(servers_file)

        # Create list that contains which locations (their ids) have been selected
        selected_loc_ids = []

        # Structure with information about the node and its unique view of the network
        # Each of this elements will be passed individual blockchain node
        nodes_view = []

        for i in range(0, args.nodes):
            idx = random.randint(0, len(pings) - 1)
            loc_peers = pings.pop(idx)
            loc_info = next(x for x in servers if x["id"] == loc_peers["id"])

            selected_loc_ids.append(loc_peers["id"])
            nodes_view.append({"loc": loc_info, "peers": loc_peers["pings"]})

        for i in range(0, len(selected_loc_ids)):
            to_delete = []
            for x in nodes_view[i]["peers"].keys():
                if x not in selected_loc_ids:
                    to_delete.append(x)

            for x in to_delete:
                del nodes_view[i]["peers"][x]

    peers_hist_labels = []
    peers_hist_values = []
    peers_nodes = []
    peers_edges = []

    # Create labels and values for histgram data provided in output
    for prob_i in range(min(num_of_peers), max(num_of_peers) + 1):
        peers_hist_labels.append(prob_i)
        peers_hist_values.append(num_of_peers.count(prob_i))

    # Add latency for each connection between nodes
    for i in range(0, total_node_count):
        peers_nodes.append({"id": i, "label": next(x for x in nodes_view if x["loc"]["id"] == selected_loc_ids[i])["loc"]["name"]})

    resolved_nodes = {}
    for i in range(0, total_node_count):
        for j in list(nx.neighbors(g, i)):
            if j in resolved_nodes:
                continue
            else:
                i_selection = next(x for x in nodes_view if x["loc"]["id"] == selected_loc_ids[i])["peers"]
                label = "-"

                for key, value in i_selection.items():
                    if key == selected_loc_ids[j]:
                        label = value
                        break

                peers_edges.append({"from": i, "to": j, "label": str(round(label, 1))})
                g.add_edge(i, j, weight=label)

                # Add connection from A -> B
                if 'connections' not in nodes_view[i]:
                    nodes_view[i]['connections'] = []
                nodes_view[i]['connections'].append({'id': selected_loc_ids[j], 'delay': label, 'p2p_id': '-'})

                # Add connection from B -> A
                if 'connections' not in nodes_view[j]:
                    nodes_view[j]['connections'] = []
                nodes_view[j]['connections'].append({'id': selected_loc_ids[i], 'delay': nodes_view[j]["peers"][selected_loc_ids[i]], 'p2p_id': '-'})
        resolved_nodes[i] = True

    # Remove 'peers' from nodes_view as it is not needed anymore, connections are already created and stored
    for i in range(0, len(nodes_view)):
        del nodes_view[i]["peers"]

    print(f"Total edges: {total_edges}")
    print(f"Possible edge drop count: {edge_drop_counter}")

    # Save output data in path from ENV (provided in deployment) or locally if not available
    with open(path_to_save, "w") as output_file:
        output_file.write(json.dumps({
            "hist_labels": peers_hist_labels,
            "hist_values": peers_hist_values,
            "nodes": peers_nodes,
            "edges": peers_edges,
            "nodes_view": nodes_view
        }))

    print("[Stats] Generating stats...")

    n = 5000
    all_hops = []
    all_tau = []
    for _ in tqdm(range(0, n)):
        hops, tau = traverse_graph(g, range(len(g.nodes)))
        all_hops.append(hops - 1)
        all_tau.append(tau / 1000)

    print('[Stats] Average number of hops:', round(sum(all_hops) / len(all_hops), 3))
    print('[Stats] Average tau:', round(sum(all_tau) / len(all_tau), 3), 'secs')
    print('[Stats] ================================================')

    print('[Stats] Min number of hops:', min(all_hops))
    print('[Stats] Min tau:', round(min(all_tau), 3), 'secs')
    print('[Stats] Max number of hops:', max(all_hops))
    print('[Stats] Max tau:', round(max(all_tau), 3), 'secs')
    print('[Stats] Standard deviation hops:', round(statistics.stdev(all_hops), 3))
    print('[Stats] Standard deviation tau:', round(statistics.stdev(all_tau), 3))


def traverse_graph(g: nx.Graph, seq) -> tuple[int, int]:
    randomPair = random.sample(seq, 2)  # Two node ids without duplicate
    startNode = randomPair[0]
    endNode = randomPair[1]

    path = nx.shortest_path(g, source=startNode, target=endNode, method='dijkstra')
    return len(list(path)), nx.path_weight(g, path=path, weight='weight')


if __name__ == "__main__":
    main()
