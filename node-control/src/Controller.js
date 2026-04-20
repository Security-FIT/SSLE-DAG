import React from 'react';
import { HOSTNAME_PORT } from './env';

export default class Controller extends React.Component {
    constructor(props) {
        super(props);

        this.state = {
            nodesConnected: '',
            numberOfTransactions: 100,
        };

        this.textFadeTimeout = null;

        this.displaySuccessfullResponse = this.displaySuccessfullResponse.bind(this);
        this.sharePublicKeys = this.sharePublicKeys.bind(this);
        this.startBlockchain = this.startBlockchain.bind(this);
        this.stopBlockchain = this.stopBlockchain.bind(this);
        this.sendTransactionPack = this.sendTransactionPack.bind(this);
    }

    componentDidMount() {
        this.retrieveNodesConnected();
    }

    async retrieveNodesConnected(showResponseText = false) {
        const nodeReadyResponse = await fetch(`http://${HOSTNAME_PORT}/commands/nodes-ready`, { method: 'GET' });
        if (nodeReadyResponse.ok) {
            const data = await nodeReadyResponse.text();
            this.setState({ nodesConnected: data });

            if (showResponseText) {
                this.displaySuccessfullResponse();
            }
        } else {
            console.error('Error fetching nodes ready:', nodeReadyResponse.statusText);
        }
    }

    async sharePublicKeys() {
        const sharePublicKeysResponse = await fetch(`http://${HOSTNAME_PORT}/commands/distribute-public-keys`, { method: 'GET' });
        if (sharePublicKeysResponse.ok) {
            this.displaySuccessfullResponse();

            document.getElementById('build-merkle-tree').classList.add('disabled');
        } else {
            console.error('Error sharing public keys:', sharePublicKeysResponse.statusText);
        }
    }

    async startBlockchain() {
        const startBlockchainResponse = await fetch(`http://${HOSTNAME_PORT}/commands/generate-and-start-blockchain`, { method: 'GET' });
        if (startBlockchainResponse.ok) {
            this.displaySuccessfullResponse();

            document.getElementById('start-blockchain').classList.add('disabled');
        } else {
            console.error('Error starting blockchain:', startBlockchainResponse.statusText);
        }
    }

    async stopBlockchain() {
        const stopBlockchainResponse = await fetch(`http://${HOSTNAME_PORT}/commands/stop-blockchain`, { method: 'GET' });
        if (stopBlockchainResponse.ok) {
            this.displaySuccessfullResponse();

            document.getElementById('start-blockchain').classList.remove('disabled');
        } else {
            console.error('Error stopping blockchain:', stopBlockchainResponse.statusText);
        }
    }

    async sendTransactionPack() {
        if (!this.checkValidIntNumber(this.state.numberOfTransactions)) {
            return;
        }
        const sendTransactionPackResponse = await fetch(`http://${HOSTNAME_PORT}/commands/send-transaction-pack/${this.state.numberOfTransactions}`, { method: 'GET' });
        if (sendTransactionPackResponse.ok) {
            this.displaySuccessfullResponse();
        } else {
            console.error('Error sending transaction pack:', sendTransactionPackResponse.statusText);
        }
    }

    displaySuccessfullResponse() {
        if (this.textFadeTimeout === null) {
            const textElem = document.getElementById('responseText');
            textElem.classList.remove('opacity-0');
            textElem.classList.add('opacity-100');

            this.textFadeTimeout = setTimeout(() => {
                textElem.classList.remove('opacity-100');
                textElem.classList.add('opacity-0');
            }, 2000);
        } else {
            clearTimeout(this.textFadeTimeout);
            this.textFadeTimeout = null;
            this.displaySuccessfullResponse();
        }
    }

    checkValidIntNumber(str) {
        return /^\d+$/.test(str);
    }

    render() {
        return (
            <div className="container">
                <div className="d-flex flex-column justify-content-center align-items-center">
                    <div>
                        <p><b>Step 1.</b> Wait until all nodes are ready</p>
                        <p><b>Step 2.</b> Notify nodes to share their public keys</p>
                        <p><b>Step 3.</b> Publish genesis the block and start the blockchain</p>
                        <p><b>At any time:</b> Send transaction packs to all nodes</p>
                    </div>
                    <div className="d-flex flex-row justify-content-center align-items-center">
                        <button className="btn btn-primary m-2" onClick={() => this.retrieveNodesConnected(true)}>
                            Refresh number of nodes ready
                        </button>
                        <p className="ms-2 mb-0">{`Connected nodes: ${this.state.nodesConnected}`}</p>
                    </div>
                    <hr className="w-100" />
                    <div className="d-flex flex-row justify-content-center align-items-center">
                        <button id="build-merkle-tree" className="btn btn-primary m-2" onClick={this.sharePublicKeys}>
                            Share public keys and build merkle tree
                        </button>
                    </div>
                    <div className="d-flex flex-row justify-content-center align-items-center">
                        <button id="start-blockchain" className="btn btn-primary m-2" onClick={this.startBlockchain}>
                            Publish genesis block and start blockchain protocol
                        </button>
                    </div>
                    <div className="d-flex flex-row justify-content-center align-items-center">
                        <button className="btn btn-primary m-2" onClick={this.sendTransactionPack}>
                            Send transaction pack
                        </button>
                        <input
                            className={'mb-3 form-control'}
                            type={'text'}
                            placeholder={'Number of transactions'}
                            value={this.state.numberOfTransactions}
                            onChange={(event) => this.setState({ numberOfTransactions: event.target.value })}
                        />
                    </div>
                    <div className="d-flex flex-row justify-content-center align-items-center">
                        <button id="stop-blockchain" className="btn btn-warning mt-5 m-2" onClick={this.stopBlockchain}>
                            Stop blockchain protocol
                        </button>
                    </div>
                    <p id="responseText" className="response-text opacity-0">
                        Request send.
                    </p>
                </div>
            </div>
        );
    }
}
