import { Logger } from '@nestjs/common';
import { readFileSync, existsSync } from 'fs';
import { NetworkConfig, NodeInfo } from 'src/commands/command.types';

export default (): any => {
    const sab = new SharedArrayBuffer(4);
    const int32 = new Int32Array(sab);

    while (!existsSync(`${process.env.VOLUME_PATH}network.json`)) {
        Logger.log('Network file not found, waiting for 5000ms...');
        Atomics.wait(int32, 0, 0, 5000);
    }

    Logger.log('Network file found, reading...');
    const config: NetworkConfig = JSON.parse(
        readFileSync(`${process.env.VOLUME_PATH}network.json`, 'utf8'),
    );

    const nodes: NodeInfo[] = [];

    for (let node_view of config.nodes_view) {
        const node: NodeInfo = {
            id: node_view.loc.id,
            name: node_view.loc.name,
            lat: node_view.loc.lat,
            lon: node_view.loc.lon,
            connections: node_view.connections,
        }
        nodes.push(node);
    }

    return {
        volumePath: process.env.VOLUME_PATH,
        nodes: nodes,
        histLabels: config.hist_labels,
        histValues: config.hist_values,
        nodesNetwork: config.nodes,
        nodeEdges: config.edges,
    };
};
