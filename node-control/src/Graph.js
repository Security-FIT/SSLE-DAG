import React from 'react';
import { HOSTNAME_PORT } from './env';

export default class Graph extends React.Component {
    constructor(props) {
        super(props);
        this.mount = false;
    }

    async componentDidMount() {
        if (!this.mount) {
            const Data = await this.retrieveNetwork();
            var vis = require('vis-network/standalone/umd/vis-network.min.js');

            const nodesCopy = [...Data.nodes];
            const edgesCopy = [...Data.edges];

            const nodes = new vis.DataSet(nodesCopy);
            const edges = new vis.DataSet(edgesCopy);
            const data = { nodes: nodes, edges: edges };

            const options = {
                nodes: {
                    shape: 'dot',
                    size: 16,
                },
                layout: {
                    randomSeed: 34,
                },
                physics: {
                    forceAtlas2Based: {
                        gravitationalConstant: -26,
                        centralGravity: 0.005,
                        springLength: 230,
                        springConstant: 0.18,
                    },
                    maxVelocity: 146,
                    solver: 'forceAtlas2Based',
                    timestep: 0.35,
                    stabilization: {
                        enabled: true,
                        iterations: 2000,
                        updateInterval: 25,
                    },
                },
            };

            const container = document.getElementById('network-visualization');
            const network = new vis.Network(container, data, options);
            network.stopSimulation();
        }
    }

    async retrieveNetwork() {
        const networkResponse = await fetch(`http://${HOSTNAME_PORT}/network`, {
            method: 'GET',
            headers: {
                'Content-Type': 'application/json',
            },
        });
        if (networkResponse.ok) {
            const data = await networkResponse.json();
            return data;
        } else {
            console.error('Error fetching network:', networkResponse.statusText);
            return null;
        }
    }

    render() {
        return (
            <div className="App">
                <div id="network-visualization" className="network"></div>
                Zoom in/out with mouse wheel or touchpad. Drag nodes to discover their neighbours.
                <br />
            </div>
        );
    }
}
