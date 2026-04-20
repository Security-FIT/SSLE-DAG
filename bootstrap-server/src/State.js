export default class State {
    constructor() {
        this.state = 'BOOTING';
    }

    getState() {
        return this.state;
    }

    incrementState() {
        switch (this.state) {
            case 'BOOTING':
                this.state = 'RECEIVE_HELLO';
                break;
            case 'RECEIVE_HELLO':
                this.state = 'SEND_CONF';
                break;
            case 'SEND_CONF':
                this.state = 'SEND_RDY';
                break;
            case 'SEND_RDY':
                this.state = 'FINISH'
                break;
            default:
                this.state = 'FINISH';
                break;
        }
        return this.state;
    }
}
